package listeners

import (
	"crypto/sha256"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	"github.com/btcsuite/btcutil/hdkeychain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bjornoj/txnotify/email"
)

func TestOnchainTx(t *testing.T) {
	t.Run("sends email when address receives new transaction", func(t *testing.T) {
		// first we initialize everything we need, and create an address
		sender := email.NewEmailSender(os.Getenv("EMAIL_PASSWORD"))
		address := MockAddress()

		// spawn the listener and add the address to the watch list
		channel := make(chan *wire.MsgTx)
		go OnchainTx(channel, sender, nil, chaincfg.RegressionNetParams)
		WatchAddress(address, Notification{Email: "bo@jalborg.com"}, gofakeit.Sentence(3), 0)
		defer func() {
			delete(WatchedAddresses, address.String())
		}()

		// create a transaction paying to the address we mocked
		pkScript, err := txscript.PayToAddrScript(address)
		require.NoError(t, err)
		var wireTx wire.MsgTx
		wireTx.AddTxOut(&wire.TxOut{
			Value:    100_000_000,
			PkScript: pkScript,
		})

		// send the transaction to the channel
		channel <- &wireTx

		require.Eventually(t, func() bool {
			return len(WatchedTxids) == 1
		}, time.Second, 10*time.Millisecond)

		require.Eventually(t, func() bool {
			return len(WatchedTxids) == 0
		}, 5*time.Second, time.Second)
	})

}

func TestWatchAddress(t *testing.T) {

	address := MockAddress()
	email := "xkifrakoi@gmail.com"
	description := gofakeit.Sentence(3)
	confirmations := int64(gofakeit.Number(0, 100))

	t.Run("can add address", func(t *testing.T) {
		require.Len(t, WatchedAddresses, 0)

		WatchAddress(address, Notification{Email: email}, description, confirmations)

		require.Len(t, WatchedAddresses, 1)
	})

	got, ok := WatchedAddresses[address.String()]
	require.True(t, ok)

	t.Run("can match on address", func(t *testing.T) {
		assert.Equal(t, email, got.Notify.Email)
	})

	t.Run("can add description", func(t *testing.T) {
		assert.Equal(t, description, got.Description)
	})

	t.Run("can add confirmations", func(t *testing.T) {
		assert.Equal(t, confirmations, got.WantConfirmations)
	})
}

func TestOnchainBlock(t *testing.T) {
	// TODO: Test deep confirmation. From 1 - 10. Also make sure stuff isn't sent out twice
	// TODO: Connect to local regtest node.. Shit, that's a large task, that I'm not ready for now.
	// first we initialize everything we need, and create an address
	sender := email.NewEmailSender(os.Getenv("EMAIL_PASSWORD"))
	address := MockAddress()

	// spawn the listener and add the address to the watch list
	channel := make(chan *wire.MsgBlock)
	go OnchainBlock(&rpcclient.Client{}, channel, chaincfg.RegressionNetParams, sender)

	confirmations := int64(gofakeit.Number(1, 10))
	WatchAddress(address, Notification{Email: "bo@jalborg.com"}, gofakeit.Sentence(3), confirmations)

	t.Run("sends out confirmation on deep confirmation", func(t *testing.T) {
	})
}

var (
	keyLock sync.Mutex
	// extendedKey is where we store our private key, see init function
	extendedKey = func() *hdkeychain.ExtendedKey {
		seed := sha256.Sum256([]byte{byte(time.Now().UnixNano())})
		key, err := hdkeychain.NewMaster(seed[:], &chaincfg.RegressionNetParams)
		if err != nil {
			panic("could not create new extended key from string")
		}

		return key
	}()
)

var addressCounter uint32 = 0

// MockAddress mocks a btc address from the extendedKey using child derivation
func MockAddress() btcutil.Address {
	address, err := extendedKey.Address(&chaincfg.RegressionNetParams)
	if err != nil {
		panic(fmt.Errorf("could not create address: %w", err))
	}

	return address
}
