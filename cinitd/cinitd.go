package main

import (
	"context"
	"encoding/json"
	"flag"
	"os"
	"sync"

	"github.com/sirupsen/logrus"
	"github.com/ulfox/cinit/cinitd/channels"
	h "github.com/ulfox/cinit/cinitd/listeners/http/server"
	"github.com/ulfox/cinit/cinitd/listeners/uds"
	"github.com/ulfox/cinit/cinitd/models"
	"github.com/ulfox/cinit/cinitd/processes"
	"github.com/ulfox/cinit/cinitd/services"
	"github.com/ulfox/cinit/cinitd/utils"
	udsc "github.com/ulfox/cinit/cli/uds"
)

var (
	sver     string
	logger   *logrus.Logger
	sockAddr string
	port     string
	listenAt string
	watchAll bool
)

func main() {
	cinitDevMode := flag.Bool("dev", false, "enable dev mode, to allow cinit to run if it is not pid 1")
	unixSocket := flag.String("unix-socket", "/tmp/cinit.sock", "cinitd unix socket")
	httpPortArg := flag.String("http-port", "8081", "cinitd http listening port")
	httpInterfaceArg := flag.String("http-listener", "127.0.0.1", "cinitd http listening interface")
	logDir := flag.String("log-dir", "/var/log/cinitd", "services logdir")
	flag.Parse()

	logger = logrus.New()

	log := logger.WithFields(logrus.Fields{
		"Component": "Cinitd",
	})

	sockAddr = *unixSocket
	port = *httpPortArg
	listenAt = *httpInterfaceArg

	env := utils.GetCInitEnv()
	if env["debug"] == "true" {
		logger.SetLevel(logrus.DebugLevel)
	}
	// if e := env["sockaddr"]; e != "" {
	// 	sockAddr = e
	// }
	if e := env["port"]; e != "" {
		port = e
	}
	if e := env["listen"]; e != "" {
		listenAt = e
	}

	if *logDir == "" {
		log.Fatal("service logdir can not be empty")
	}

	if !(*cinitDevMode) {
		watchAll = true
		log.Warnf("Watchall is enabled. On stop cinitd will send SIGTERM and SIGKILL (on timeout) to all processes")
	}

	cpid := os.Getpid()
	if cpid != 1 && !(*cinitDevMode) {
		log.Fatalf("not pid 1, exiting...")
		return
	}

	log.Info("Initiated")

	sysSigs := utils.NewOSSignal()
	prcSigStop := make(chan bool)
	soSigStop := make(chan bool)
	remoteChan := channels.NewRemoteChannel(60)
	serviceChan := channels.NewServiceChannel(60, 60)

	// Async Groups
	var processOperatorWaitGroup sync.WaitGroup
	var serviceOperatorWaitGroup sync.WaitGroup
	var unixServerWaitGroup sync.WaitGroup
	var httpServerWaitGroup sync.WaitGroup

	serviceOperator := services.NewProcessOperator(
		soSigStop,
		logger,
	)
	go serviceOperator.Init(remoteChan, serviceChan, &serviceOperatorWaitGroup)
	serviceOperator.Ready()

	processOperator := processes.NewProcessOperator(
		prcSigStop,
		logger,
		watchAll,
		serviceChan,
		*logDir,
	)
	go processOperator.Init(&processOperatorWaitGroup)
	processOperator.Ready()

	udsServerCtx, udsServerCancel := context.WithCancel(context.Background())
	unixServer := uds.NewServerFactory(udsServerCtx, remoteChan, sockAddr, logger, &unixServerWaitGroup)
	unixServer.ListenBackground()

	httpServerCtx, httpServerCancel := context.WithCancel(context.Background())
	httpServer := h.NewServerFactory(httpServerCtx, remoteChan, port, listenAt, logger, &httpServerWaitGroup)
	httpServer.ListenBackground()

	sysSigs.Wait()
	log.Infof("Interrupted")

	udsServerCancel()

	// empty message will cause the server to loop over and catch the canceled context
	go func() {
		dataBytes, err := json.Marshal(models.Service{T: "shutdown"})
		if err != nil {
			log.Fatal(err)
		}

		client := udsc.NewClientFactory(sockAddr, logger)
		err = client.OpenCom()
		if err != nil {
			log.Fatal(err)
		}

		err = client.SendByteBatch(dataBytes)
		if err != nil {
			log.Fatal(err)
		}
		client.CloseCom()

	}()

	unixServerWaitGroup.Wait()

	httpServerCancel()
	httpServerWaitGroup.Wait()

	soSigStop <- true
	serviceOperatorWaitGroup.Wait()
	close(soSigStop)

	prcSigStop <- true
	processOperatorWaitGroup.Wait()
	close(prcSigStop)

	sysSigs.Close()
	remoteChan.Close()
	serviceChan.Close()

	log.Infof("Bye!")
}
