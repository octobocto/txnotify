package email

import (
	"net/smtp"

	"github.com/sirupsen/logrus"
)

var log = logrus.New()

const (
	from = "alerts@txnotify.com"
)

type EmailSender struct {
	password string
}

func NewEmailSender(password string) EmailSender {
	return EmailSender{password: password}
}

func (e EmailSender) Send(to string, subject, message string) error {

	msg := "From: " + from + "\n" +
		"To: " + to + "\n" +
		"Subject: " + subject + "\n\n" +
		message

	err := smtp.SendMail("smtp.gmail.com:587",
		smtp.PlainAuth("", from, e.password, "smtp.gmail.com"),
		from, []string{to}, []byte(msg))
	if err != nil {
		log.Printf("smtp error: %s", err)
		return err
	}

	log.WithField("to", to).Info("sent email")

	return nil
}
