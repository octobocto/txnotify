package db

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDB_MigrateUp(t *testing.T) {
	db, err := New(nil)
	require.NoError(t, err)

	require.NoError(t, db.MigrateUp())
}
