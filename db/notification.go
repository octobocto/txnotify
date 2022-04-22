package db

import (
	"fmt"

	"github.com/google/uuid"
)

type Notification struct {
	ID            uuid.UUID `db:"id"`
	UserID        uuid.UUID `db:"user_id"`
	Identifier    string    `db:"identifier"`
	Confirmations uint32    `db:"confirmations"`
	Email         string    `db:"email"`
	Description   string    `db:"description"`
}

func (n Notification) Save(database *DB) (Notification, error) {
	var id uuid.UUID
	rows, err := database.NamedQuery("INSERT INTO notifications (user_id, identifier, confirmations, email, description)"+
		"VALUES (:user_id, :identifier, :confirmations, :email, :description) RETURNING id", n)
	if err != nil {
		return Notification{}, err
	}
	next := rows.Next()
	if !next {
		return Notification{}, fmt.Errorf("could not insert notification")
	}
	if err := rows.Scan(&id); err != nil {
		return Notification{}, fmt.Errorf("could not scan into struct: %w", err)
	}

	n.ID = id
	return n, nil
}

func ListNotifications(database *DB, userID uuid.UUID) ([]Notification, error) {
	var notifications []Notification

	err := database.Select(&notifications, `SELECT * FROM notifications where user_id = $1`, userID)
	if err != nil {
		return nil, err
	}

	return notifications, nil
}

func GetNotification(database *DB, ID uuid.UUID) (Notification, error) {
	var notification Notification
	return notification, database.Get(&notification, `SELECT * FROM notifications WHERE id = $1`, ID)
}
