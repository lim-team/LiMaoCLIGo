package client

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSendMessage(t *testing.T) {
	c := New("tcp://127.0.0.1:6666", WithUID("1"), WithToken("1234"))
	err := c.Connect()
	assert.NoError(t, err)
}
