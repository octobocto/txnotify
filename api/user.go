package api

import (
	"context"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/bjornoj/txnotify/db"
	"github.com/bjornoj/txnotify/email"
	rpc "github.com/bjornoj/txnotify/proto"
)

type userService struct {
	database *db.DB
	network  chaincfg.Params
	btc      *rpcclient.Client
	sender   email.EmailSender

	rpc.UnsafeUserServer
}

func NewUserService(database *db.DB, network chaincfg.Params, btc *rpcclient.Client, sender email.EmailSender) userService {
	return userService{
		database: database,
		network:  network,
		btc:      btc,
		sender:   sender,
	}
}

var _ rpc.UserServer = userService{}

var log = logrus.New()

type User struct {
	ID uuid.UUID `db:"id"`
}

func (u userService) CreateUser(_ context.Context, _ *rpc.CreateUserRequest) (*rpc.CreateUserResponse, error) {

	user, err := createUser(u.database)
	if err != nil {
		return nil, err
	}

	log.WithField("id", user.ID).Info("created new user")

	return &rpc.CreateUserResponse{Id: user.ID.String()}, nil
}

func createUser(database *db.DB) (User, error) {
	var id uuid.UUID
	err := database.QueryRow("INSERT INTO users DEFAULT VALUES RETURNING id").Scan(&id)
	if err != nil {
		return User{}, err
	}

	return User{ID: id}, nil
}
