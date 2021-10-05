package commands

import (
	"fmt"

	"github.com/ulfox/cinit/cinitd/models"
)

func (c *Command) Action(name *string, T string) error {
	if name == nil {
		return c.wrapErr("Service needs to be set")
	}

	if len(*name) < 1 {
		return c.wrapErr("Service name can not be empty")
	}

	service := models.Service{
		T:    T,
		Name: *name,
	}

	data, err := c.pushToServer(service)
	if err != nil {
		return c.wrapErr(err)
	}

	fmt.Println(string(data))

	// var prettyJSON bytes.Buffer
	// err = json.Indent(&prettyJSON, data, "", "\t")
	// if err != nil {
	// 	fmt.Print(string(data))
	// 	return nil
	// }

	// c.logger.Info("\n", prettyJSON.String())

	return nil
}
