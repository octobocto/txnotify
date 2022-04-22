package api

import (
	"context"
	"testing"

	"github.com/bjornoj/txnotify/db"
	"github.com/brianvoe/gofakeit/v6"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gotest.tools/assert"

	rpc "github.com/bjornoj/txnotify/proto"
)

func TestNotifyService_CreateNotification(t *testing.T) {

	t.Run("can create notification", func(t *testing.T) {

	})

	t.Run("specifying neither address nor txid returns error", func(t *testing.T) {

	})

	t.Run("inserts notification into database", func(t *testing.T) {

	})
}

func TestNotification_Save(t *testing.T) {
	user := createUserTest(t)

	identifier := gofakeit.Sentence(5)
	confirmations := gofakeit.Uint32()
	email := gofakeit.Email()
	description := gofakeit.JobDescriptor()

	notif, err := db.Notification{
		UserID:        user.ID,
		Identifier:    identifier,
		Confirmations: confirmations,
		Email:         email,
		Description:   description,
	}.Save(testDB)
	require.NoError(t, err)

	assert.Equal(t, identifier, notif.Identifier)
	assert.Equal(t, confirmations, notif.Confirmations)
	assert.Equal(t, email, notif.Email)
	assert.Equal(t, description, notif.Description)

	t.Run("can not save without user", func(t *testing.T) {
		_, err := db.Notification{
			UserID:        user.ID,
			Identifier:    identifier,
			Confirmations: confirmations,
			Email:         email,
			Description:   description,
		}.Save(testDB)
		require.Error(t, err)
	})

	t.Run("can not save with non-existent user", func(t *testing.T) {
		_, err := db.Notification{
			UserID:        uuid.New(),
			Identifier:    identifier,
			Confirmations: confirmations,
			Email:         email,
			Description:   description,
		}.Save(testDB)
		require.Error(t, err)
	})
}

func TestGetNotification(t *testing.T) {
	user := createUserTest(t)

	identifier := gofakeit.Sentence(5)
	confirmations := gofakeit.Uint32()
	email := gofakeit.Email()
	description := gofakeit.JobDescriptor()

	notif, err := db.Notification{
		UserID:        user.ID,
		Identifier:    identifier,
		Confirmations: confirmations,
		Email:         email,
		Description:   description,
	}.Save(testDB)
	require.NoError(t, err)

	gotNotif, err := db.GetNotification(testDB, notif.ID)
	require.NoError(t, err)

	assert.DeepEqual(t, notif, gotNotif)
}

func TestListNotifications(t *testing.T) {
	user := createUserTest(t)

	length := gofakeit.Number(3, 10)
	for i := 0; i < length; i++ {
		_, err := db.Notification{
			UserID:        user.ID,
			Identifier:    gofakeit.BitcoinAddress(),
			Confirmations: gofakeit.Uint32(),
			Email:         gofakeit.Email(),
			Description:   gofakeit.JobDescriptor(),
		}.Save(testDB)
		require.NoError(t, err)
	}

	service := notifyService{database: testDB}
	notifs, err := service.ListNotifications(context.Background(), &rpc.ListNotificationsRequest{UserId: user.ID.String()})
	require.NoError(t, err)

	require.Len(t, notifs.Notifications, length)

	for _, notif := range notifs.Notifications {
		require.NotEmpty(t, notif.UserId)
		require.NotEmpty(t, notif.Email)
		require.NotEmpty(t, notif.Description)
		require.NotEmpty(t, notif.Confirmations)
		require.NotEmpty(t, notif.Identifier)
	}
}

func createUserTest(t *testing.T) User {
	user, err := createUser(testDB)
	require.NoError(t, err)

	return user
}
