package database

import (
	"testing"

	"github.com/dimspell/gladiator/internal/app/logger"
	"github.com/stretchr/testify/assert"
)

func TestMigrate(t *testing.T) {
	logger.SetDiscardLogger()

	db, err := NewMemory()
	if err != nil {
		assert.NoError(t, err)
		return
	}
	assert.NoError(t, db.Ping())
}
