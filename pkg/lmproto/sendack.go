package lmproto

import (
	"fmt"

	"github.com/pkg/errors"
)

// SendackPacket 发送回执包
type SendackPacket struct {
	Framer
	MessageID   int64      // 消息ID（全局唯一）
	MessageSeq  uint32     // 消息序列号（用户唯一，有序）
	ClientSeq   uint64     // 客户端序列号 (客户端提供，服务端原样返回)
	ClientMsgNo string     // 客户端消息编号(目前只有mos协议有效)
	ReasonCode  ReasonCode // 原因代码
}

// GetPacketType 包类型
func (s *SendackPacket) GetPacketType() PacketType {
	return SENDACK
}
func (s *SendackPacket) String() string {
	return fmt.Sprintf("MessageSeq:%d MessageId:%d ReasonCode:%s", s.MessageSeq, s.MessageID, s.ReasonCode)
}

func decodeSendack(frame Frame, data []byte, version uint8) (Frame, error) {
	dec := NewDecoder(data)
	sendackPacket := &SendackPacket{}
	sendackPacket.Framer = frame.(Framer)
	var err error
	var clientSeq uint32
	if clientSeq, err = dec.Uint32(); err != nil {
		return nil, errors.Wrap(err, "解码ClientSeq失败！")
	}
	sendackPacket.ClientSeq = uint64(clientSeq)
	if sendackPacket.MessageID, err = dec.Int64(); err != nil {
		return nil, errors.Wrap(err, "解码MessageId失败！")
	}
	if sendackPacket.MessageSeq, err = dec.Uint32(); err != nil {
		return nil, errors.Wrap(err, "解码MessageSeq失败！")
	}

	// 原因代码
	var reasonCode uint8
	if reasonCode, err = dec.Uint8(); err != nil {
		return nil, errors.Wrap(err, "解码ChannelType失败！")
	}
	sendackPacket.ReasonCode = ReasonCode(reasonCode)

	return sendackPacket, err
}

func encodeSendack(frame Frame, version uint8) ([]byte, error) {
	sendackPacket := frame.(*SendackPacket)
	enc := NewEncoder()
	enc.WriteUint32(uint32(sendackPacket.ClientSeq))
	// 消息唯一ID
	enc.WriteInt64(sendackPacket.MessageID)
	// 消息序列号(客户端维护)
	enc.WriteUint32(sendackPacket.MessageSeq)
	// 原因代码
	enc.WriteUint8(sendackPacket.ReasonCode.Byte())
	return enc.Bytes(), nil
}
