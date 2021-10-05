package main

import (
	"flag"
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/ulfox/cinit/cinitd/utils"
	"github.com/ulfox/cinit/cli/commands"
)

var (
	logger *logrus.Logger
	host   string
)

func main() {
	cinitdPort := flag.String("cinitd-port", "8081", "listening port of cinitd")
	cinitdHost := flag.String("cinitd-host", "localhost", "cinitd host")
	cinitService := flag.String("f", "", "service file")
	serviceRegister := flag.Bool("register", false, "create a new service")
	serviceDelete := flag.Bool("delete", false, "delete a service, this will also stop the service")
	serviceStop := flag.Bool("stop", false, "stop a service")
	serviceStatus := flag.Bool("status", false, "get the status of a registed service")
	serviceName := flag.String("name", "", "service name")
	serviceStart := flag.Bool("start", false, "start a service")
	serviceList := flag.Bool("list", false, "list services")

	flag.Parse()

	logger = logrus.New()

	if *cinitdHost == "" || *cinitdPort == "" {
		logger.Fatalf("cinitd host/port can not be empty")
	}

	host = *cinitdHost
	if *cinitdPort != "80" {
		host = fmt.Sprintf("%s:%s", *cinitdHost, *cinitdPort)
	}

	env := utils.GetCInitEnv()
	if env["debug"] != "true" {
		logger.SetLevel(logrus.DebugLevel)
	}

	c := commands.NewCommandFactory(host, logger)

	if *serviceRegister {
		if *cinitService == "" {
			logger.Fatal("Service file can not be empty")
		}

		err := c.ReadService(*cinitService)
		if err != nil {
			logger.Fatal(err)
		}
		err = c.RegisterService()
		if err != nil {
			logger.Fatal(err)
		}
	} else if *serviceStatus {
		err := c.Action(serviceName, "status")
		if err != nil {
			logger.Fatal(err)
		}
	} else if *serviceDelete {
		err := c.Action(serviceName, "delete")
		if err != nil {
			logger.Fatal(err)
		}
	} else if *serviceStop {
		err := c.Action(serviceName, "stop")
		if err != nil {
			logger.Fatal(err)
		}
	} else if *serviceStart {
		err := c.Action(serviceName, "start")
		if err != nil {
			logger.Fatal(err)
		}
	} else if *serviceList {
		sn := "all"
		err := c.Action(&sn, "list")
		if err != nil {
			logger.Fatal(err)
		}
	}
}
