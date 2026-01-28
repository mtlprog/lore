package sync

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewSyncer(t *testing.T) {
	t.Run("nil pool returns error", func(t *testing.T) {
		syncer, err := New(nil, "https://horizon.stellar.org")
		assert.Nil(t, syncer)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database pool")
	})

	t.Run("empty horizon URL returns error", func(t *testing.T) {
		// This will also fail on nil pool, but we check for empty URL first in current impl
		syncer, err := New(nil, "")
		assert.Nil(t, syncer)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "horizon URL")
	})
}

func TestNewRepository(t *testing.T) {
	t.Run("nil pool returns error", func(t *testing.T) {
		repo, err := NewRepository(nil)
		assert.Nil(t, repo)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database pool")
	})
}

func TestDefaultFailureThreshold(t *testing.T) {
	assert.Equal(t, 0.1, DefaultFailureThreshold)
}
