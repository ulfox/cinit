package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/sirupsen/logrus"
	e "github.com/ulfox/cinit/cinitd/errors"
	"github.com/ulfox/cinit/cinitd/models"
	h "github.com/ulfox/cinit/cli/http"
	"gopkg.in/yaml.v2"
)

type erf = func(e interface{}, p ...interface{}) error

type Service struct {
	Name    string `yaml:"name"`
	Command string `yaml:"command"`
	Args    string `yaml:"args,omitempty"`
}

type Command struct {
	logger  *logrus.Logger
	host    string
	wrapErr erf
	service Service
}

func NewCommandFactory(host string, l *logrus.Logger) *Command {
	return &Command{
		logger:  l,
		host:    host,
		wrapErr: e.WrapErr,
	}
}

func (c *Command) pushToServer(service models.Service) ([]byte, error) {
	dataBytes, err := json.Marshal(service)
	if err != nil {
		return nil, c.wrapErr(err)
	}

	client, err := h.NewClientFactory(fmt.Sprintf("http://%s/api/services", c.host), c.logger)
	if err != nil {
		return nil, c.wrapErr(err)
	}

	data, err := client.SendByteBatch(dataBytes)
	if err != nil {
		return nil, c.wrapErr(err)
	}

	return data, nil
}

func (c *Command) ReadService(file string) error {
	f, err := os.Open(file)
	if err != nil {
		return c.wrapErr(err)
	}

	data := make([]byte, 0)
	block := make([]byte, 512)
	for {
		b, err := f.Read(block)
		if err != nil {
			if err != io.EOF {
				return c.wrapErr(err)
			}
			break
		}

		data = append(data, block[:b]...)
	}
	f.Close()

	c.service = Service{}
	err = yaml.Unmarshal(data, &(c.service))
	if err != nil {
		return c.wrapErr(err)
	}

	return nil
}
