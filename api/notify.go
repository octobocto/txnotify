package api

import (
	"context"
	"fmt"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/google/uuid"

	"github.com/bjornoj/txnotify/db"
	"github.com/bjornoj/txnotify/email"
	"github.com/bjornoj/txnotify/listeners"
	rpc "github.com/bjornoj/txnotify/proto"
)

type notifyService struct {
	network  chaincfg.Params
	btc      *rpcclient.Client
	sender   email.EmailSender
	database *db.DB

	rpc.UnsafeNotifyServer
}

func NewNotifyService(database *db.DB, network chaincfg.Params, btc *rpcclient.Client, sender email.EmailSender) notifyService {
	return notifyService{
		database: database,
		network:  network,
		btc:      btc,
		sender:   sender,
	}
}

var _ rpc.NotifyServer = notifyService{}

func (n notifyService) CreateNotification(ctx context.Context, req *rpc.Notification) (*rpc.CreateNotificationResponse, error) {
	body := fmt.Sprintf(`New notification registered
email: %s
txid/address: %s
confirmations: %d`, req.Email, req.Identifier, req.Confirmations)

	_ = n.sender.Send("bo@jalborg.com", fmt.Sprintf("New notification created to %s", req.Email), body)

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, err
	}

	err = listeners.WatchIdentifier(&n.network, req.Identifier, listeners.Notification{
		Email:       req.Email,
		SlackURL:    req.SlackWebhookUrl,
		CallbackURL: req.CallbackUrl,
	}, req.Description, int64(req.Confirmations))
	if err != nil {
		return nil, fmt.Errorf("could not register new notification: %w", err)
	}

	notification, err := db.Notification{
		UserID:        userID,
		Identifier:    req.Identifier,
		Confirmations: req.Confirmations,
		Email:         req.Email,
		Description:   req.Description,
	}.Save(n.database)
	if err != nil {
		return nil, err
	}

	return &rpc.CreateNotificationResponse{
		Id: notification.ID.String(),
	}, nil
}

func (n notifyService) ListNotifications(ctx context.Context, req *rpc.ListNotificationsRequest) (*rpc.ListNotificationsResponse, error) {

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, fmt.Errorf("invalid userID: %v", userID)
	}

	notifications, err := db.ListNotifications(n.database, userID)
	if err != nil {
		return nil, err
	}

	var notifs []*rpc.Notification
	for _, notification := range notifications {
		notifs = append(notifs, &rpc.Notification{
			UserId:        notification.UserID.String(),
			Identifier:    notification.Identifier,
			Confirmations: notification.Confirmations,
			Email:         notification.Email,
			Description:   notification.Description,
		})
	}

	return &rpc.ListNotificationsResponse{
		Notifications: notifs,
	}, nil
}
