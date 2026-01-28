package handler

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewHandler(t *testing.T) {
	t.Run("nil stellar service returns error", func(t *testing.T) {
		h, err := New(nil, nil, nil)
		assert.Nil(t, h)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "stellar service")
	})
}
