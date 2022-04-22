package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path"
	"runtime/debug"
	"strings"
	"time"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcd/wire"
	grpcmiddleware "github.com/grpc-ecosystem/go-grpc-middleware"
	recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/lightninglabs/gozmq"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"

	"github.com/bjornoj/txnotify/api"
	"github.com/bjornoj/txnotify/db"
	"github.com/bjornoj/txnotify/email"
	"github.com/bjornoj/txnotify/listeners"
	rpc "github.com/bjornoj/txnotify/proto"
)

var log = logrus.New()

func main() {

	app := cli.NewApp()
	app.Name = "txnotify"
	app.Version = "0.1"
	app.Usage = "Managing helper for developing lightning payment processor"
	app.EnableBashCompletion = true
	commands := []*cli.Command{
		Serve(),
	}

	app.Commands = commands

	err := app.Run(os.Args)
	if err != nil {
		// only print error if something was supplied to txnotify, help
		// message is printed anyways
		if len(os.Args) > 1 {
			_, _ = fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(1)
	}
	// we need to explicitly exit, otherwise we'll hang forever waiting for a signal
	os.Exit(0)

}

func Serve() *cli.Command {
	serve := &cli.Command{
		Name:  "serve",
		Usage: "Starts TXNotify",
		Action: func(c *cli.Context) error {

			var database *db.DB
			var err error
			for i := 0; i < 10; i++ {
				database, err = db.New(c)
				if err != nil {
					fmt.Println(fmt.Errorf("could not create new db: %w", err))
					time.Sleep(time.Second)
					continue
				}
				break
			}
			if err != nil {
				return err
			}

			bitcoin, err := newBitcoinConnection(c)
			if err != nil {
				return err
			}

			emailSender := email.NewEmailSender(c.String("email-password"))

			grpcServer := grpc.NewServer(UnaryServerInterceptor())
			rpc.RegisterNotifyServer(grpcServer, api.NewNotifyService(database, bitcoin.network, bitcoin.btcctl, emailSender))
			rpc.RegisterUserServer(grpcServer, api.NewUserService(database, bitcoin.network, bitcoin.btcctl, emailSender))

			server := Server{
				database:   database,
				grpcServer: grpcServer,
				bitcoind:   bitcoin,
			}

			restMux, err := server.registerRESTServiceHandlers()
			if err != nil {
				return err
			}

			httpHandler := splitGrpcAndRest(server.grpcServer, restMux)

			httpHandler = h2c.NewHandler(httpHandler, &http2.Server{})
			httpHandler = allowCORS(httpHandler)
			httpHandler = removeTrailingSlash(httpHandler)

			server.httpServer = &http.Server{
				Addr:    "0.0.0.0:9002",
				Handler: httpHandler,
			}

			log.Info("listening on localhost:9002")

			err = bitcoin.StartZmq(bitcoin.btcctl, ZmqConfig{
				Transactions: c.Int("bitcoind.zmqpubrawtx"),
				Blocks:       c.Int("bitcoind.zmqpubrawblock"),
			}, bitcoin.network, emailSender)
			if err != nil {
				return err
			}

			return server.httpServer.ListenAndServe()
		},
		Flags: []cli.Flag{
			// database flags start here
			&cli.StringFlag{
				Name:  "db.host",
				Usage: "Host the database runs on",
				Value: "0.0.0.0",
			},
			&cli.IntFlag{
				Name:  "db.port",
				Usage: "Port database runs on",
				Value: 5432,
			},

			// bitcoind flags start here
			&cli.StringFlag{
				Name:     "bitcoind.rpcuser",
				Usage:    "The bitcoind RPC username",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "bitcoind.rpcpassword",
				Usage:    "The bitcoind RPC password",
				Required: true,
			},
			&cli.IntFlag{
				Name:  "bitcoind.rpcport",
				Usage: "The bitcoind RPC port",
			},
			&cli.StringFlag{
				Name:  "bitcoind.rpchost",
				Usage: "The bitcoind RPC host",
				Value: "localhost",
			},
			&cli.IntFlag{
				Name:     "bitcoind.zmqpubrawblock",
				Usage:    "The port listening for ZMQ connections to deliver raw block notifications",
				Required: true,
			},
			&cli.IntFlag{
				Name:     "bitcoind.zmqpubrawtx",
				Usage:    "The port listening for ZMQ connections to deliver raw transaction notifications",
				Required: true,
			},
			&cli.StringFlag{
				Name:  "network",
				Usage: "the network lnd is running on e.g. mainnet, testnet, etc.",
				Value: "regtest",
			},

			// util flags
			&cli.StringFlag{
				Name:  "email-password",
				Usage: "Email password for the email account specified in the code",
			},
		},
	}

	return serve
}

// Server is the server for txnotify. It serves the API over both REST and gRPC.
type Server struct {
	database   *db.DB
	grpcServer *grpc.Server
	httpServer *http.Server // server HTTP and gRPC over the same port

	// TODO: Add database
	bitcoind BitcoinConn
}

func (s *Server) registerRESTServiceHandlers() (http.Handler, error) {
	type registerFunc = func(context.Context, *runtime.ServeMux, string, []grpc.DialOption) error

	registerFuncs := []registerFunc{
		rpc.RegisterNotifyHandlerFromEndpoint,
		rpc.RegisterUserHandlerFromEndpoint,
	}

	grpcMux := runtime.NewServeMux()
	ctx := context.Background()
	for _, register := range registerFuncs {
		if err := register(ctx, grpcMux, "0.0.0.0:9002", []grpc.DialOption{grpc.WithInsecure()}); err != nil {
			return nil, err
		}
	}

	mux := http.NewServeMux()
	// serve gRPC REST gateway under /
	mux.Handle("/", grpcMux)

	return mux, nil
}

var corsHeaders = strings.Join([]string{
	"Content-Type", "Accept",
	"Authorization", "Access-Control-Allow-Origin",
}, ",")

// allowCORS allows Cross Origin Resource Sharing from any origin.
func allowCORS(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if origin := r.Header.Get("Origin"); origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			if r.Method == "OPTIONS" && r.Header.Get("Access-Control-Request-Method") != "" {
				w.Header().Set("Access-Control-Allow-Headers", corsHeaders)

				methods := []string{"GET", "HEAD", "POST", "PUT", "DELETE"}
				w.Header().Set("Access-Control-Allow-Methods", strings.Join(methods, ","))
				return
			}
		}
		h.ServeHTTP(w, r)
	})
}

func removeTrailingSlash(h http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/" && strings.HasSuffix(request.URL.Path, "/") {
			request.URL.Path = strings.TrimSuffix(request.URL.Path, "/")
		}

		h.ServeHTTP(writer, request)
	})
}

// returns a http.Handler that delegates to grpcServer on incoming gRPC
// connections or restHandler otherwise. Copied from cockroachdb.
func splitGrpcAndRest(grpcServer *grpc.Server, restHandler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// This is a partial recreation of gRPC's internal checks https://github.com/grpc/grpc-go/pull/514/files#diff-95e9a25b738459a2d3030e1e6fa2a718R61
		if r.ProtoMajor == 2 && strings.Contains(r.Header.Get("Content-Type"), "application/grpc") {
			grpcServer.ServeHTTP(w, r)
		} else {
			restHandler.ServeHTTP(w, r)
		}
	})
}

// bitcoinConfig contains everything we need to reliably start a bitcoind node.
type bitcoinConfig struct {
	RpcPort  int
	RpcHost  string
	P2pPort  int
	User     string
	Password string
	// Network is the network we're running on
	Network chaincfg.Params
}

// ToConnConfig converts this BitcoindConfig to the format the rpcclient
// library expects.
func (conf *bitcoinConfig) ToConnConfig() *rpcclient.ConnConfig {
	host := conf.RpcHost
	if host == "" {
		host = "127.0.0.1"
	}

	log.WithFields(logrus.Fields{
		"host":    conf.RpcHost,
		"user":    conf.User,
		"network": conf.Network.Name,
		"p2pport": conf.P2pPort,
		"rpcport": conf.RpcPort,
	}).Info("converting config to rpc config")

	return &rpcclient.ConnConfig{
		Host:         fmt.Sprintf("%s:%d", host, conf.RpcPort),
		User:         conf.User,
		Pass:         conf.Password,
		DisableTLS:   true, // Bitcoin Core doesn't do TLS
		HTTPPostMode: true, // Bitcoin Core only supports HTTP POST mode
	}
}

// readNetwork reads the network flag, erroring if an invalid value is passed
func readNetwork(c *cli.Context) (chaincfg.Params, error) {
	var network chaincfg.Params
	networkString := c.String("network")
	switch networkString {
	case "mainnet":
		network = chaincfg.MainNetParams
	case "testnet", "testnet3":
		network = chaincfg.TestNet3Params
	case "regtest", "":
		network = chaincfg.RegressionNetParams
	default:
		return chaincfg.Params{}, fmt.Errorf("unknown network: %s. Valid: mainnet, testnet, regtest", networkString)
	}
	return network, nil
}

// Conn represents a persistent client connection to a bitcoind node
// that listens for events read from a ZMQ connection
type BitcoinConn struct {
	// btcctl is a bitcoind rpc connection
	btcctl *rpcclient.Client

	// zmqBlockConn is the ZMQ connection we'll use to read raw block
	// events
	zmqBlockConn *gozmq.Conn
	// zmqTxConn is the ZMQ connection we'll use to read raw new
	// transaction events
	zmqTxConn *gozmq.Conn

	// config is the config used for this connection
	config bitcoinConfig
	// network is the network this cnonection is running on
	network chaincfg.Params
}

// ZmqConfig contains what we need to connect to bitcoind ZMQ channels
type ZmqConfig struct {
	Transactions int
	Blocks       int
}

// StartZmq attempts to establish a ZMQ connection to a bitcoind node. If
// successful, a goroutine is spawned to read events from the ZMQ connection.
// It's possible for this function to fail due to a limited number of connection
// attempts. This is done to prevent waiting forever on the connection to be
// established in the case that the node is down.
// Blocks and txs is the URLs to connect to for block and transaction messages, respectively
func (c *BitcoinConn) StartZmq(btc *rpcclient.Client, config ZmqConfig, network chaincfg.Params,
	sender email.EmailSender) error {
	const timeout = time.Second
	var err error

	// Establish two different ZMQ connections to bitcoind to retrieve block
	// and transaction event notifications. We'll use two as a separation of
	// concern to ensure one type of event isn't dropped from the connection
	// queue due to another type of event filling it up.

	blockUrl := fmt.Sprintf("tcp://%s:%d", c.config.RpcHost, config.Blocks)
	c.zmqBlockConn, err = gozmq.Subscribe(blockUrl, []string{"rawblock"}, timeout)
	if err != nil {
		return fmt.Errorf("gozmq.Subscribe rawblock: %w", err)
	}

	txUrl := fmt.Sprintf("tcp://%s:%d", c.config.RpcHost, config.Transactions)
	c.zmqTxConn, err = gozmq.Subscribe(txUrl, []string{"rawtx"}, timeout)
	if err != nil {
		if err := c.zmqBlockConn.Close(); err != nil {
			log.WithError(err).Error("could not close ZMQ block connection", err)
		}
		return fmt.Errorf("gozmq.Subscribe rawtx: %w", err)
	}

	zmqTxCh := make(chan *wire.MsgTx)
	zmqBlockCh := make(chan *wire.MsgBlock)

	go c.blockEventHandler(zmqBlockCh)
	go c.txEventHandler(zmqTxCh)

	go listeners.OnchainBlock(btc, zmqBlockCh, network, sender)
	go listeners.OnchainTx(zmqTxCh, sender, btc, network)

	return nil
}

// blockEventHandler reads raw blocks events from the ZMQ block socket and
// forwards them to the channel registered on the Conn
//
// NOTE: This must be run as a goroutine.
func (c *BitcoinConn) blockEventHandler(channel chan<- *wire.MsgBlock) {
	log.Info("started listening for ZMQ block notifications")

	for {
		// Poll an event from the ZMQ socket. This is where the goroutine
		// will hang until new messages are received
		var bufs [][]byte
		msgBytes, err := c.zmqBlockConn.Receive(bufs)
		if err != nil {
			// EOF should only be returned if the connection was
			// explicitly closed, so we can exit at this point.
			if err == io.EOF {
				return
			}

			// It's possible that the connection to the socket
			// continuously times out, so we'll prevent logging this
			// error to prevent spamming the logs.
			netErr, ok := err.(net.Error)
			if ok && netErr.Timeout() {
				continue
			}

			if !strings.Contains(err.Error(), "cannot receive from a closed connection") {
				log.WithError(err).Error("Unable to receive ZMQ rawblock message")
			}

			return
		}

		// We have an event! We'll now ensure it is a block event,
		// deserialize it, and report it to the zmq block channel
		// the other end is (hopefully) listening at
		eventType := string(msgBytes[0])
		switch eventType {
		case "rawblock":
			block := &wire.MsgBlock{}
			r := bytes.NewReader(msgBytes[1])
			if err := block.Deserialize(r); err != nil {
				log.WithError(err).Error("Unable to deserialize block")
				continue
			}

			log.WithField("hash", block.BlockHash()).Trace("received new block")
			// send the deserialized block to the block channel
			channel <- block

		default:
			// It's possible that the message wasn't fully read if
			// bitcoind shuts down, which will produce an unreadable
			// event type. To prevent from logging it, we'll make
			// sure it conforms to the ASCII standard.
			if eventType == "" {
				continue
			}

			log.WithField("eventType", eventType).Warn("Received unexpected event type from rawblock subscription")
		}
	}
}

// txEventHandler reads raw blocks events from the ZMQ block socket and
// forwards them to the zmqTxCh found in the Conn
//
// NOTE: This must be run as a goroutine.
func (c *BitcoinConn) txEventHandler(channel chan<- *wire.MsgTx) {
	log.Info("started listening for ZMQ transaction notifications")

	for {
		// Poll an event from the ZMQ socket
		var bufs [][]byte
		msgBytes, err := c.zmqTxConn.Receive(bufs)
		if err != nil {
			// EOF should only be returned if the connection was
			// explicitly closed, so we can exit at this point.
			if err == io.EOF {
				return
			}

			// It's possible that the connection to the socket
			// continuously times out, so we'll prevent logging this
			// error to prevent spamming the logs.
			netErr, ok := err.(net.Error)
			if ok && netErr.Timeout() {
				continue
			}

			if strings.Contains(err.Error(), "cannot receive from a closed connection") {
				continue
			}

			log.WithError(err).Error("Unable to receive ZMQ rawblock message")

			return
		}

		// We have an event! We'll now ensure it is a transaction event,
		// deserialize it, and report it to the different rescan
		// clients.
		eventType := string(msgBytes[0])
		switch eventType {
		case "rawtx":
			tx := &wire.MsgTx{}
			r := bytes.NewReader(msgBytes[1])
			// Deserialize the bytes from reader r into tx
			if err := tx.Deserialize(r); err != nil {
				log.WithError(err).Error("Unable to deserialize transaction")
				continue
			}

			// send the tx event to the channel
			channel <- tx

		default:
			// It's possible that the message wasn't fully read if
			// bitcoind shuts down, which will produce an unreadable
			// event type. To prevent from logging it, we'll make
			// sure it conforms to the ASCII standard.
			if eventType == "" {
				continue
			}

			log.WithField("event", eventType).Warn("Received unexpected event type from rawtx subscription")
		}

	}
}

// awaitBitcoind tries to get a RPC response from bitcoind, returning an error
// if that isn't possible within a set of attempts
func awaitBitcoind(conn *BitcoinConn) error {
	log := log.WithFields(logrus.Fields{
		"host": conn.config.RpcHost,
		"port": conn.config.RpcPort,
		"user": conn.config.User,
	})

	var err error
	const attempts = 10
	for i := 0; i < attempts; i++ {
		_, err = conn.btcctl.GetNetworkInfo()
		switch {
		case err == nil:
			return nil

			// invalid credentials, no point in continuing
		case strings.Contains(err.Error(), "status code: 401"):
			return errors.New("invalid bitcoind credentials")

		default:
			err = fmt.Errorf("awaitBitcoind(%+v): %w", conn, err)
		}

		log.WithField("attempt", i).WithError(err).Error("tried connecting to bitcoin")

		time.Sleep(time.Second)
	}

	log.WithError(err).Error("error is")
	return err
}

func newBitcoinConnection(c *cli.Context) (BitcoinConn, error) {
	network, err := readNetwork(c)
	if err != nil {
		return BitcoinConn{}, err
	}

	conf := bitcoinConfig{
		RpcHost:  c.String("bitcoind.rpchost"),
		RpcPort:  c.Int("bitcoind.rpcport"),
		Password: c.String("bitcoind.rpcpassword"),
		User:     c.String("bitcoind.rpcuser"),
		Network:  network,
	}

	if conf.RpcPort == 0 {
		switch conf.Network.Name {
		case chaincfg.MainNetParams.Name:
			conf.RpcPort = 8332
		case chaincfg.TestNet3Params.Name:
			conf.RpcPort = 18332
		case chaincfg.RegressionNetParams.Name:
			conf.RpcPort = 18443
		default:
			return BitcoinConn{}, errors.New("network is not set")
		}
	}

	client, err := rpcclient.New(conf.ToConnConfig(), nil)
	if err != nil {
		return BitcoinConn{}, fmt.Errorf("could not create new bitcoind rpcclient,"+
			"is bitcoind running? %w", err)
	}

	conn := &BitcoinConn{
		btcctl:  client,
		config:  conf,
		network: conf.Network,
	}

	if err = awaitBitcoind(conn); err != nil {
		return BitcoinConn{}, err
	}

	log.Info("successfully connected to bitcoind")

	return *conn, nil
}

// UnaryServerInterceptor creates the standard server interceptor, along with any custom interceptors given.
func UnaryServerInterceptor() grpc.ServerOption {

	baseInterceptors := []grpc.UnaryServerInterceptor{
		recovery.UnaryServerInterceptor(recovery.WithRecoveryHandlerContext(func(ctx context.Context, p interface{}) (
			err error) {
			log.WithField("panic", p).Error("Panicked when serving gRPC, dumping stack trace")
			log.Error(string(debug.Stack()))

			return fmt.Errorf("internal server error: %w", err)
		})),
		func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
			log := log.WithFields(
				logrus.Fields{
					"grpc.service": path.Dir(info.FullMethod)[1:],
					"grpc.method":  path.Base(info.FullMethod),
				})

			if d, ok := ctx.Deadline(); ok {
				log = log.WithFields(
					logrus.Fields{
						"grpc.request.deadline": d.Format(time.RFC3339),
					})
			}

			resp, err := handler(ctx, req)

			log = log.WithField("body", req)

			if err == nil {
				log.Info("successful request")
			} else {
				log.WithError(err).Info("failed request")
			}

			return resp, err
		},
	}

	// because base grpc only allows one UnaryInterceptor, we use the
	// grpc_middleware package to add support for multiple interceptors
	return grpcmiddleware.WithUnaryServerChain(baseInterceptors...)
}
