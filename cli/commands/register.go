package commands

import (
	"fmt"
	"regexp"

	"github.com/ulfox/cinit/cinitd/models"
)

func (c *Command) RegisterService() error {
	if c.service.Name == "" {
		return c.wrapErr("Service needs to be set")
	}

	if c.service.Command == "" {
		return c.wrapErr("Command needs to be set")
	}

	service := models.Service{
		T:       "register",
		Name:    c.service.Name,
		Command: c.service.Command,
	}

	if len(c.service.Args) > 0 {
		service.Args = regexp.MustCompile(`[^\s"]+|"([^"]*)"`).FindAllString(c.service.Args, -1)
	}

	data, err := c.pushToServer(service)
	if err != nil {
		return c.wrapErr(err)
	}

	fmt.Println(string(data))

	return nil
}
