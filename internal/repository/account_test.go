package repository

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewAccountRepository(t *testing.T) {
	t.Run("nil pool returns error", func(t *testing.T) {
		repo, err := NewAccountRepository(nil)
		assert.Nil(t, repo)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database pool is required")
	})
}
