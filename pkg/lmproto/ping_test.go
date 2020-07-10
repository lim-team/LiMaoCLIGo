package lmproto

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPingEncodeAndDecode(t *testing.T) {
	packet := &PingPacket{}

	codec := New()
	// 编码
	packetBytes, err := codec.EncodePacket(packet, 1)
	assert.NoError(t, err)

	// 解码
	resultPacket, _, err := codec.DecodePacket(packetBytes, 1)
	assert.NoError(t, err)
	_, ok := resultPacket.(*PingPacket)
	assert.Equal(t, true, ok)
}
