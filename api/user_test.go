package api

import (
	"context"
	"testing"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bjornoj/txnotify/db"
	"github.com/bjornoj/txnotify/email"
	rpc "github.com/bjornoj/txnotify/proto"
)
var testDB = db.NewTest("api_test")

func TestUserService_CreateUser(t *testing.T) {

	 service := NewUserService(testDB, chaincfg.RegressionNetParams, nil, email.EmailSender{})

	 user, err := service.CreateUser(context.Background(), &rpc.CreateUserRequest{})

	 require.NoError(t, err)
	 assert.NotEmpty(t, user.Id)
}