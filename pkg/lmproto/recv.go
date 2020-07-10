package lmproto

import (
	"fmt"

	"github.com/pkg/errors"
)

// RecvPacket 收到消息的包
type RecvPacket struct {
	Framer
	MessageID   int64  // 服务端的消息ID(全局唯一)
	MessageSeq  uint32 // 消息序列号 （用户唯一，有序递增）
	ClientMsgNo string // 客户端唯一标示
	Timestamp   int32  // 服务器消息时间戳(10位，到秒)
	FromUID     string // 发送者UID
	ChannelID   string // 频道ID
	ChannelType uint8  // 频道类型
	Payload     []byte // 消息内容
}

// GetPacketType 获得包类型
func (r *RecvPacket) GetPacketType() PacketType {
	return RECV
}
func (r *RecvPacket) String() string {
	return fmt.Sprintf("Recv Header:%s MessageID:%d MessageSeq:%d Timestamp:%d FromUid:%s ChannelID:%s ChannelType:%d Payload:%s", r.Framer, r.MessageID, r.MessageSeq, r.Timestamp, r.FromUID, r.ChannelID, r.ChannelType, string(r.Payload))
}

func decodeRecv(frame Frame, data []byte, version uint8) (Frame, error) {
	dec := NewDecoder(data)
	recvPacket := &RecvPacket{}
	recvPacket.Framer = frame.(Framer)
	var err error
	// 消息全局唯一ID
	if recvPacket.MessageID, err = dec.Int64(); err != nil {
		return nil, errors.Wrap(err, "解码MessageId失败！")
	}
	// 消息序列号 （用户唯一，有序递增）
	if recvPacket.MessageSeq, err = dec.Uint32(); err != nil {
		return nil, errors.Wrap(err, "解码MessageSeq失败！")
	}
	if version > 1 {
		// 客户端唯一标示
		if recvPacket.ClientMsgNo, err = dec.String(); err != nil {
			return nil, errors.Wrap(err, "解码ClientMsgNo失败！")
		}
	}
	// 消息时间
	if recvPacket.Timestamp, err = dec.Int32(); err != nil {
		return nil, errors.Wrap(err, "解码Timestamp失败！")
	}
	// 频道ID
	if recvPacket.ChannelID, err = dec.String(); err != nil {
		return nil, errors.Wrap(err, "解码ChannelId失败！")
	}
	// 频道类型
	if recvPacket.ChannelType, err = dec.Uint8(); err != nil {
		return nil, errors.Wrap(err, "解码ChannelType失败！")
	}
	// 发送者
	if recvPacket.FromUID, err = dec.String(); err != nil {
		return nil, errors.Wrap(err, "解码FromUID失败！")
	}
	payloadStartLen := 8 + 4 + 4 + uint32(len(recvPacket.ChannelID)+2) + 1 + uint32(len(recvPacket.FromUID)+2) // 消息ID长度 + 消息序列号长度 + 消息时间长度 +频道ID长度+字符串标示长度 + 频道类型长度 + 发送者uid长度
	if version > 1 {
		payloadStartLen += uint32(len(recvPacket.ClientMsgNo) + 2)
	}
	if uint32(len(data)) < payloadStartLen {
		return nil, errors.New("解码RECV消息时失败！payload开始长度位置大于整个剩余数据长度！")
	}
	recvPacket.Payload = data[payloadStartLen:]
	return recvPacket, err
}

func encodeRecv(frame Frame, version uint8) ([]byte, error) {
	recvPacket := frame.(*RecvPacket)
	enc := NewEncoder()
	// 消息唯一ID
	enc.WriteInt64(recvPacket.MessageID)
	// 消息有序ID
	enc.WriteUint32(recvPacket.MessageSeq)
	if version > 1 {
		// 客户端唯一标示
		enc.WriteString(recvPacket.ClientMsgNo)
	}
	// 消息时间戳
	enc.WriteInt32(recvPacket.Timestamp)
	// 频道ID
	enc.WriteString(recvPacket.ChannelID)
	// 频道类型
	enc.WriteUint8(recvPacket.ChannelType)
	// 发送者
	enc.WriteString(recvPacket.FromUID)
	// 消息内容
	enc.WriteBytes(recvPacket.Payload)
	return enc.Bytes(), nil
}
