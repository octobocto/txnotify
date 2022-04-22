package email

import (
	"os"
	"testing"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/stretchr/testify/require"
)

func TestSend(t *testing.T) {

	t.Run("can send email", func(t *testing.T) {
		sender := NewEmailSender(os.Getenv("EMAIL_PASSWORD"))

		require.NoError(t, sender.Send(gofakeit.Email(), "New Transaction", "new tx!"))
	})
}
