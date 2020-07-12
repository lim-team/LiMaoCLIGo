package client

import (
	"errors"
	"log"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/lim-team/LiMaoCLIGo/pkg/lmproto"
	"github.com/lim-team/LiMaoCLIGo/pkg/util"
	"go.uber.org/atomic"
)

var defaultOpts *Options

func init() {
	defaultOpts = NewOptions()
	// flagSet := lmFlagSet(defaultOpts)
	// flagSet.Parse(os.Args[1:])
}

// OnRecv 收到消息事件
type OnRecv func(recv *lmproto.RecvPacket) error

// OnClose 连接关闭
type OnClose func()

// Client 狸猫客户端
type Client struct {
	opts              *Options              // 狸猫IM配置
	sending           []*lmproto.SendPacket // 发送中的包
	proto             *lmproto.LiMaoProto
	addr              string      // 连接地址
	connected         atomic.Bool // 是否已连接
	conn              net.Conn
	heartbeatTimer    *time.Timer // 心跳定时器
	stopHeartbeatChan chan bool
	retryPingCount    int // 重试ping次数
	clientIDGen       atomic.Uint64
	onRecv            OnRecv
	onClose           OnClose
	sendTotalMsgBytes atomic.Int64 // 发送消息总bytes数
}

// New 创建客户端
func New(addr string, opts ...Option) *Client {
	for _, opt := range opts {
		if opt != nil {
			if err := opt(defaultOpts); err != nil {
				panic(err)
			}
		}
	}
	return &Client{
		opts:              defaultOpts,
		addr:              addr,
		sending:           make([]*lmproto.SendPacket, 0),
		proto:             lmproto.New(),
		heartbeatTimer:    time.NewTimer(time.Second * 20),
		stopHeartbeatChan: make(chan bool, 0),
	}
}

// Connect 连接到IM
func (c *Client) Connect() error {
	network, address, _ := parseAddr(c.addr)
	var err error
	c.conn, err = net.Dial(network, address)
	if err != nil {
		return err
	}
	err = c.sendPacket(&lmproto.ConnectPacket{
		Version:         c.opts.ProtoVersion,
		DeviceFlag:      lmproto.WEB,
		ClientTimestamp: time.Now().Unix(),
		UID:             c.opts.UID,
		Token:           c.opts.Token,
	})
	if err != nil {
		return err
	}
	f, err := c.proto.DecodePacketWithConn(c.conn, c.opts.ProtoVersion)
	if err != nil {
		return err
	}
	connack, ok := f.(*lmproto.ConnackPacket)
	if !ok {
		return errors.New("返回包类型有误！不是连接回执包！")
	}
	if connack.ReasonCode != lmproto.ReasonSuccess {
		return errors.New("连接失败！")
	}
	if len(c.sending) > 0 {
		for _, packet := range c.sending {
			c.sendPacket(packet)
		}
	}
	go c.loopConn()
	go c.loopPing()
	return nil
}

// Disconnect 断开IM
func (c *Client) Disconnect() {
	c.handleClose()
}

func (c *Client) handleClose() {
	if c.connected.Load() {
		c.connected.Store(false)
		c.conn.Close()
		if c.onClose != nil {
			c.onClose()
		}
	}
}

// SendMessage 发送消息
func (c *Client) SendMessage(channel *Channel, payload []byte) (*lmproto.SendackPacket, error) {
	packet := &lmproto.SendPacket{
		ClientSeq:   c.clientIDGen.Add(1),
		ClientMsgNo: util.GenUUID(),
		ChannelID:   channel.ChannelID,
		ChannelType: channel.ChannelType,
		Payload:     payload,
	}
	c.sending = append(c.sending, packet)
	err := c.sendPacket(packet)
	if err != nil {
		return nil, err
	}
	f, err := c.proto.DecodePacketWithConn(c.conn, c.opts.ProtoVersion)
	if err != nil {
		return nil, err
	}
	return f.(*lmproto.SendackPacket), err
}

// SetOnRecv 设置收消息事件
func (c *Client) SetOnRecv(onRecv OnRecv) {
	c.onRecv = onRecv
}

// SetOnClose 设置关闭事件
func (c *Client) SetOnClose(onClose OnClose) {
	c.onClose = onClose
}

// GetSendMsgBytes 获取已发送字节数
func (c *Client) GetSendMsgBytes() int64 {
	return c.sendTotalMsgBytes.Load()
}

func (c *Client) loopPing() {
	for {
		select {
		case <-c.heartbeatTimer.C:
			if c.retryPingCount >= 3 {
				c.conn.Close() // 如果重试三次没反应就断开连接，让其重连
				return
			}
			c.ping()
			c.retryPingCount++
			break
		case <-c.stopHeartbeatChan:
			goto exit
		}
	}
exit:
}

func (c *Client) ping() {
	c.sendPacket(&lmproto.PingPacket{})
}

// 发送包
func (c *Client) sendPacket(packet lmproto.Frame) error {
	data, err := c.proto.EncodePacket(packet, c.opts.ProtoVersion)
	if err != nil {
		return err
	}
	c.sendTotalMsgBytes.Add(int64(len(data)))
	_, err = c.conn.Write(data)
	return err
}

func (c *Client) loopConn() {
	for {
		frame, err := c.proto.DecodePacketWithConn(c.conn, c.opts.ProtoVersion)
		if err != nil {
			log.Println("解码数据失败！", err)
			c.handleClose()
			goto exit
		}
		c.handlePacket(frame)
	}
exit:
	log.Println("断开，开始重连...")
	c.connected.Store(true)
	c.stopHeartbeatChan <- true
	c.Connect()
}

func (c *Client) handlePacket(frame lmproto.Frame) {
	switch frame.GetPacketType() {
	case lmproto.SENDACK: // 发送回执
		c.handleSendackPacket(frame.(*lmproto.SendackPacket))
		break
	case lmproto.RECV: // 收到消息
		c.handleRecvPacket(frame.(*lmproto.RecvPacket))
		break
	}
}

func (c *Client) handleSendackPacket(packet *lmproto.SendackPacket) {
	for i, sendPacket := range c.sending {
		if sendPacket.ClientSeq == packet.ClientSeq {
			c.sending = append(c.sending[:i], c.sending[i+1:]...)
			break
		}
	}
}

// 处理接受包
func (c *Client) handleRecvPacket(packet *lmproto.RecvPacket) {
	var err error
	if c.onRecv != nil {
		err = c.onRecv(packet)
	}
	if err == nil {
		c.sendPacket(&lmproto.RecvackPacket{
			MessageID:  packet.MessageID,
			MessageSeq: packet.MessageSeq,
		})
	}
}

func parseAddr(addr string) (network, address string, port int) {
	network = "tcp"
	address = strings.ToLower(addr)
	if strings.Contains(address, "://") {
		pair := strings.Split(address, "://")
		network = pair[0]
		address = pair[1]
		pair2 := strings.Split(address, ":")
		portStr := pair2[1]
		portInt64, _ := strconv.ParseInt(portStr, 10, 64)
		port = int(portInt64)
	}
	return
}

// Channel Channel
type Channel struct {
	ChannelID   string
	ChannelType uint8
}

// NewChannel 创建频道
func NewChannel(channelID string, channelType uint8) *Channel {
	return &Channel{
		ChannelID:   channelID,
		ChannelType: channelType,
	}
}
