package listeners

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/bjornoj/txnotify/email"
)

var log = logrus.New()

func WatchIdentifier(network *chaincfg.Params, identifier string, to Notification, description string, wantConfirmations int64) error {
	address, err := btcutil.DecodeAddress(identifier, network)
	if err != nil {
		err := AddTXFromString(identifier, wantConfirmations, to, description)
		if err != nil {
			return errors.New("Identifier was neither a bitcoin address or a bitcoin txid.")
		}
		return nil
	}

	WatchAddress(address, to, description, wantConfirmations)
	return nil
}

type AddressWatch struct {
	ID                uuid.UUID
	Notify            Notification
	WantConfirmations int64
	Description       string
}

var (
	mu sync.Mutex
	// WatchedAddresses is a map connecting bitcoin addresses to emails
	WatchedAddresses = make(map[string]AddressWatch) // map[bitcoin address]email
)

func WatchAddress(address btcutil.Address, to Notification, description string, wantConfirmations int64) {
	mu.Lock()
	defer mu.Unlock()

	log.WithField("address", address.String()).Info("starting to watch address")

	addr := AddressWatch{
		ID:                uuid.New(),
		Notify:            to,
		WantConfirmations: wantConfirmations,
		Description:       description,
	}
	WatchedAddresses[address.String()] = addr
}

// OnchainTx checks if a transaction is being watched
func OnchainTx(zmqTxs <-chan *wire.MsgTx, sender email.EmailSender, btc *rpcclient.Client,
	network chaincfg.Params) {

	for {
		tx := <-zmqTxs
		txid := tx.TxHash()

		log := log.WithFields(
			logrus.Fields{
				"txid": txid.String(),
			})

		// To listen for deposits, we loop through every output of
		// the tx, and check if any of the addresses exists in our database
		for vout, output := range tx.TxOut {
			log = log.WithField("vout", vout)

			_, addresses, _, err := txscript.ExtractPkScriptAddrs(output.PkScript, &network)
			if err != nil {
				// we don't log anything here, as all non standard TXs would fail
				// this step, cluttering our logs
				continue
			}

			for _, address := range addresses {
				watchedAddress, ok := WatchedAddresses[address.String()]
				if ok {
					err := WatchTX(TxWatch{
						txid:              txid,
						notify:            watchedAddress.Notify,
						wantConfirmations: watchedAddress.WantConfirmations,
						description:       watchedAddress.Description,
					})
					if err != nil {
						log.WithError(err).Error("could not add tx")
					}
				}
			}

			err = handleNewTX(sender, vout, btcutil.Amount(output.Value))
			if err != nil {
				log.WithError(err).Error("could not send email")
				continue
			}

		}
	}
}

// TODO: Create feature that sends email on a single tx confirmation
//  It should double check that the transaction is not already confirmed

// OnchainBlock checks if a block contains a transaction we're watching.
func OnchainBlock(btc *rpcclient.Client, zmqBlocks <-chan *wire.MsgBlock, network chaincfg.Params,
	sender email.EmailSender) {

	for {
		rawBlock := <-zmqBlocks
		hash := rawBlock.Header.BlockHash()

		header, err := btc.GetBlockHeaderVerbose(&hash)
		if err != nil {
			log.WithError(err).Error("Could not query bitcoind for block")
			continue
		}

		log := log.WithField("blockHeight", header.Height)

		for _, tx := range rawBlock.Transactions {
			txid := tx.TxHash()

			confirmTxIfExists(txid, int64(header.Height))
		}

		// we handle deep wantConfirmations after the block just in case some transactions
		// were first seen in the fresh block
		err = handleNewBlock(int64(header.Height), sender)
		if err != nil {
			log.WithError(err).Error("could not handle deep confirmation")
		}

	}
}

func confirmTxIfExists(hash chainhash.Hash, height int64) {
	tx, ok := WatchedTxids[hash.String()]
	if !ok {
		return
	}

	log.WithFields(logrus.Fields{
		"txid": hash.String(),
	}).Info("found relevant transaction in new block")

	// TODO O: Write in email address received new transaction

	tx.confirmedAtBlock = &height
	WatchedTxids[hash.String()] = tx
}

type Notification struct {
	Email       string
	SlackURL    string
	CallbackURL string
}

type TxWatch struct {
	ID uuid.UUID

	txid chainhash.Hash
	// notify contains different ways of contacting the user
	notify Notification
	// if set, it means the transaction is confirmed.
	confirmedAtBlock *int64
	// how many wantConfirmations this transaction wants before a notification is sent
	wantConfirmations int64
	// description is set by the user.
	description string
}

var (
	txidMu sync.Mutex
	// WatchedTxids is a map connecting txids to contact info. This is the only thing that should be
	// responsible for sending out emails
	WatchedTxids = make(map[string]TxWatch)
)

func WatchTX(tx TxWatch) error {

	log.WithField("txid", tx.txid.String()).Info("starting to watch txid")

	txidMu.Lock()
	defer txidMu.Unlock()

	tx.ID = uuid.New()
	WatchedTxids[tx.txid.String()] = tx

	return nil
}

func AddTXFromString(txidString string, wantConfirmations int64, to Notification, description string) error {

	txid, err := chainhash.NewHashFromStr(txidString)
	if err != nil {
		return fmt.Errorf("txid not a valid txid: %w", err)
	}

	return WatchTX(TxWatch{
		txid:              *txid,
		notify:            to,
		wantConfirmations: wantConfirmations,
		description:       description,
	})
}

func handleNewBlock(height int64, sender email.EmailSender) error {

	for _, tx := range WatchedTxids {
		if tx.confirmedAtBlock == nil {
			return nil
		}

		notifyAtHeight := *tx.confirmedAtBlock + tx.wantConfirmations - 1 // current block is 1 confirmation, so we negate 1
		if height >= notifyAtHeight {
			log.WithFields(logrus.Fields{
				"txid":               tx.txid.String(),
				"wantConfirmations":  tx.wantConfirmations,
				"txConfirmedAtBlock": *tx.confirmedAtBlock,
			}).Info("found confirmed tx")

			SendTxConfirmed(sender, tx)
			// only delete if notification was successful
			delete(WatchedTxids, tx.txid.String())
		}
	}

	return nil
}

// handleNewTX makes sure to send notifications for every new TX
func handleNewTX(sender email.EmailSender, vout int, amount btcutil.Amount) error {

	for _, tx := range WatchedTxids {
		// if we get here it means we just got a new tx that isn't confirmed yet. Sooo we only care about txs that are
		// 0-conf here. That means new deposits to addresses.

		if tx.wantConfirmations != 0 {
			return nil
		}

		err := SendAddressReceivedTransaction(sender, tx.notify, tx.description, tx.txid, vout, amount)
		if err != nil {
			log.WithError(err).Error("could not send address received transaction email")
			return err
		}

		// only delete if notification was successful
		delete(WatchedTxids, tx.txid.String())
	}

	return nil
}

func SendAddressReceivedTransaction(sender email.EmailSender, to Notification, description string,
	txid chainhash.Hash, vout int, amount btcutil.Amount) error {

	log := log.WithFields(logrus.Fields{
		"email":       to.Email,
		"txid":        txid,
		"vout":        vout,
		"callbackURL": to.CallbackURL,
		"slackURL":    to.SlackURL,
	})

	if err := sendAddressReceivedTransactionEmail(sender, to.Email, description, txid, vout, amount); err != nil {
		log.Info("could not send email")
	}
	// TODO
	// if err := postCallback( sender, to.callbackURL, description, txid, vout, amount); err != nil {
	// 	log.Info("could not post callback")
	// }
	// if err := postSlack( sender, to.slackURL, description, txid, vout, amount); err != nil {
	// 	log.Info("could not post slack")
	// }
	return nil
}

func sendAddressReceivedTransactionEmail(sender email.EmailSender, to, description string,
	txid chainhash.Hash, vout int, amount btcutil.Amount) error {

	if to == "" {
		return nil
	}

	body := fmt.Sprintf(`Address received new transaction with
txid: %s
vout: %d
amount: %f BTC`, txid.String(), vout, amount.ToBTC())
	if description != "" {
		body += fmt.Sprintf("\ndescription: %s", description)
	}

	return sender.Send(to, "Address received transaction", body)
}

func sendTxConfirmedEmail(sender email.EmailSender, tx TxWatch) error {
	if tx.notify.Email == "" {
		return nil
	}

	if tx.confirmedAtBlock == nil {
		return errors.New("expected tx to be confirmed")
	}
	body := fmt.Sprintf(`Transaction confirmed
txid: %s
confirmed in block: %d
confirmations: %d`, tx.txid.String(), *tx.confirmedAtBlock, tx.wantConfirmations)

	if tx.description != "" {
		body += fmt.Sprintf("\ndescription: %s", tx.description)
	}

	return sender.Send(tx.notify.Email, "Transaction was confirmed", body)
}

func postCallback(tx TxWatch) error {

	if tx.notify.CallbackURL == "" {
		return nil
	}

	body := map[string]interface{}{
		"id":               tx.ID,
		"confirmedAtBlock": tx.confirmedAtBlock,
		"txid":             tx.txid,
		"description":      tx.description,
		"confirmations":    tx.wantConfirmations,
	}
	marshalledBody, err := json.Marshal(body)
	if err != nil {
		log.WithError(err).Error("Could not marshal callback payload")
		return err
	}
	log := log.WithField("body", string(marshalledBody))

	const contentType = "application/json"

	reader := bytes.NewReader(marshalledBody)
	// nolint gosec
	res, err := http.Post(tx.notify.CallbackURL, contentType, reader)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		msg := fmt.Sprintf("callback response: %s", res.Status)
		body, err := ioutil.ReadAll(res.Body)
		if err == nil {
			msg += fmt.Sprintf(": %s", string(body))
		}
		return errors.New(msg)
	}

	log.Info("posted callback")

	return nil
}

func postSlack(tx TxWatch) error {

	if tx.notify.SlackURL == "" {
		return nil
	}

	// check out https://api.slack.com/block-kit for how you can format the posted message. It's quite extensive!
	type innerSlackBlock struct {
		Type string `json:"type,omitempty"`
		Text string `json:"text,omitempty"`
	}
	type slackBlock struct {
		Type string           `json:"type,omitempty"`
		Text *innerSlackBlock `json:"text,omitempty"`
	}

	type slackFormat struct {
		Blocks []slackBlock `json:"blocks,omitempty"`
		Text   string       `json:"text,omitempty"`
	}

	body := map[string]interface{}{
		"id":               tx.ID,
		"confirmedAtBlock": tx.confirmedAtBlock,
		"txid":             tx.txid,
		"description":      tx.description,
		"confirmations":    tx.wantConfirmations,
	}
	marshalledBody, err := json.MarshalIndent(body, "\t", "")
	if err != nil {
		log.WithError(err).Error("Could not marshal callback payload")
		return err
	}

	data, err := json.Marshal(&slackFormat{
		Blocks: []slackBlock{
			{
				Type: "header",
				Text: &innerSlackBlock{
					Type: "plain_text",
					Text: "Transaction confirmed",
				},
			},
			{Type: "divider"},
			{
				Type: "section",
				Text: &innerSlackBlock{
					Type: "mrkdwn",
					Text: string(marshalledBody),
				},
			},
		},
	})
	if err != nil {
		return err
	}

	request, err := http.NewRequest("POST", tx.notify.SlackURL, bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")

	// we don't really care about the response. Fire and forget!
	res, err := http.DefaultClient.Do(request)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		bodyBytes, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return fmt.Errorf("could not extract data from body: %w", err)
		}

		return fmt.Errorf("could not post slack notification: %s", string(bodyBytes))
	}

	return nil
}

func SendTxConfirmed(sender email.EmailSender, tx TxWatch) {
	log := log.WithFields(logrus.Fields{
		"email":       tx.notify.Email,
		"txid":        tx.txid,
		"ID":          tx.ID,
		"callbackURL": tx.notify.CallbackURL,
		"slackURL":    tx.notify.SlackURL,
	})

	if err := sendTxConfirmedEmail(sender, tx); err != nil {
		log.Info("could not send email")
	}
	if err := postCallback(tx); err != nil {
		log.Info("could not post callback")
	}
	if err := postSlack(tx); err != nil {
		log.Info("could not post slack")
	}
}
