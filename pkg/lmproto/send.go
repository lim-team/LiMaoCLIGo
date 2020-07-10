package lmproto

import (
	"fmt"

	"github.com/pkg/errors"
)

// SendPacket 发送包
type SendPacket struct {
	Framer
	ClientSeq   uint64 // 客户端提供的序列号，在客户端内唯一
	ClientMsgNo string // 客户端消息唯一编号一般是uuid，为了去重
	ChannelID   string // 频道ID（如果是个人频道ChannelId为个人的UID）
	ChannelType uint8  // 频道类型（1.个人 2.群组）
	Payload     []byte // 消息内容
}

// GetPacketType 包类型
func (s *SendPacket) GetPacketType() PacketType {
	return SEND
}

func (s *SendPacket) String() string {
	return fmt.Sprintf("ClientSeq:%d ClientMsgNo:%s ChannelId:%s ChannelType:%d Payload:%s", s.ClientSeq, s.ClientMsgNo, s.ChannelID, s.ChannelType, string(s.Payload))
}

func decodeSend(frame Frame, data []byte, version uint8) (Frame, error) {
	dec := NewDecoder(data)
	sendPacket := &SendPacket{}
	sendPacket.Framer = frame.(Framer)
	var err error
	// 消息序列号(客户端维护)
	var clientSeq uint32
	if clientSeq, err = dec.Uint32(); err != nil {
		return nil, errors.Wrap(err, "解码ClientSeq失败！")
	}
	sendPacket.ClientSeq = uint64(clientSeq)
	if version > 1 {
		// // 客户端唯一标示
		if sendPacket.ClientMsgNo, err = dec.String(); err != nil {
			return nil, errors.Wrap(err, "解码ClientMsgNo失败！")
		}
	}
	// 频道ID
	if sendPacket.ChannelID, err = dec.String(); err != nil {
		return nil, errors.Wrap(err, "解码ChannelId失败！")
	}
	// 频道类型
	if sendPacket.ChannelType, err = dec.Uint8(); err != nil {
		return nil, errors.Wrap(err, "解码ChannelType失败！")
	}
	payloadStartLen := 4 + uint32(len(sendPacket.ChannelID)+2) + 1 // 消息序列号长度+频道ID长度+字符串标示长度 + 频道类型长度
	if version > 1 {
		payloadStartLen += uint32(len(sendPacket.ClientMsgNo) + 2)
	}
	if uint32(len(data)) < payloadStartLen {
		return nil, errors.New("解码SEND消息时失败！payload开始长度位置大于整个剩余数据长度！")
	}
	sendPacket.Payload = data[payloadStartLen:]
	return sendPacket, err
}

func encodeSend(frame Frame, version uint8) ([]byte, error) {
	sendPacket := frame.(*SendPacket)
	enc := NewEncoder()
	// 消息序列号(客户端维护)
	enc.WriteUint32(uint32(sendPacket.ClientSeq))
	if version > 1 {
		// 客户端唯一标示
		enc.WriteString(sendPacket.ClientMsgNo)
	}
	// 频道ID
	enc.WriteString(sendPacket.ChannelID)
	// 频道类型
	enc.WriteUint8(sendPacket.ChannelType)
	// 消息内容
	enc.WriteBytes(sendPacket.Payload)
	return enc.Bytes(), nil
}
