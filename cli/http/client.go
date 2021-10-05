package http

import (
	"bytes"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	e "github.com/ulfox/cinit/cinitd/errors"
)

type erf = func(e interface{}, p ...interface{}) error

var wrapErr erf = e.WrapErr

// Client for managing a HTTP Client
type Client struct {
	sync.Mutex
	logger   *logrus.Logger
	net      *http.Client
	endpoint string
}

// NewClientFactory for creating a new Client
func NewClientFactory(e string, l *logrus.Logger) (*Client, error) {
	caller := &Client{
		logger: l,
	}

	if !strings.HasPrefix(e, "http://") && !strings.HasPrefix(e, "https://") {
		return nil, wrapErr("missing protocol from endpoint %s", e)
	}

	caller.endpoint = e

	httpClient := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			Dial: (&net.Dialer{
				Timeout:   1 * time.Second,
				KeepAlive: 30 * time.Second,
			}).Dial,
			TLSHandshakeTimeout: 5 * time.Second,
			IdleConnTimeout:     60 * time.Second,
		},
		Timeout: 65 * time.Second,
	}

	caller.net = httpClient

	return caller, nil
}

func (c *Client) SendByteBatch(d []byte) ([]byte, error) {
	req, err := http.NewRequest("POST", c.endpoint, bytes.NewBuffer(d))
	if err != nil {
		return nil, wrapErr(err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.net.Do(req)
	if err != nil {
		return nil, wrapErr(err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, wrapErr(err)
	}

	return body, nil
}
