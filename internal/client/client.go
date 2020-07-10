package client

import "github.com/lim-team/LiMaoCLIGo/pkg/lmproto"

// Client 狸猫客户端
type Client struct {
}

// NewClient 创建客户端
func NewClient(addr string, opts ...Option) *Client {
	return &Client{}
}

// Connect 连接到IM
func (c *Client) Connect() {

}

// SendMessage 发送消息
func (c *Client) SendMessage(packet *lmproto.SendPacket) {
}

// Disconnect 断开IM
func (c *Client) Disconnect() {

}
