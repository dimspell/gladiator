package database

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMigrate(t *testing.T) {
	db, err := NewMemory()
	if err != nil {
		assert.NoError(t, err)
		return
	}
	assert.NoError(t, db.Ping())
}
