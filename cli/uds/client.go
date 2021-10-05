package uds

import (
	"fmt"
	"net"
	"sync"

	"github.com/sirupsen/logrus"
	e "github.com/ulfox/cinit/cinitd/errors"
)

type erf = func(e interface{}, p ...interface{}) error

var wrapErr erf = e.WrapErr

// Client for managing a Unix Domain Socket client
type Client struct {
	sync.Mutex
	logger     *logrus.Logger
	net        net.Conn
	unixSocket string
}

// NewClientFactory for creating a new Client
func NewClientFactory(u string, l *logrus.Logger) *Client {
	return &Client{
		unixSocket: u,
		logger:     l,
	}
}

// OpenCom for connecting to UDS
func (c *Client) OpenCom() error {
	if c.ClientIsSet() {
		return fmt.Errorf("client already set")
	}

	client, err := net.Dial("unix", c.unixSocket)
	if err != nil {
		return wrapErr(err)
	}

	c.Lock()
	c.net = client
	c.Unlock()

	return nil
}

// CloseCom for closing UDS channel
func (c *Client) CloseCom() error {
	if !c.ClientIsSet() {
		return nil
	}

	c.Lock()
	err := c.net.Close()
	c.Unlock()

	return wrapErr(err)
}

// ClientIsSet to check if UDS channel is open
func (c *Client) ClientIsSet() bool {
	c.Lock()
	client := c.net
	c.Unlock()
	return client != nil
}

// SendByteBatch for sending bytes to UDS channel
func (c *Client) SendByteBatch(b []byte) error {
	if !c.ClientIsSet() {
		return fmt.Errorf("client not set")
	}
	l, err := c.net.Write(b)
	if err != nil {
		return wrapErr(err)
	}
	c.logger.Infof("Sent %d bytes to cinitd", l)
	return nil
}
