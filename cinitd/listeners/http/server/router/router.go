package router

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/ulfox/cinit/cinitd/channels"
)

// Service for creating a new http router
type Service struct {
	Router *mux.Router
	rcmd   *channels.Remote
	logger *logrus.Logger
}

// UpdateRoutes method updates main route with our Handle Functions
func (s *Service) UpdateRoutes() *Service {
	s.Router.HandleFunc("/api/services", s.services).Methods("POST")
	return s
}

// NewRouter factory for creating a Service router
func NewRouter(rcmd *channels.Remote, l *logrus.Logger) *Service {
	return &Service{
		Router: mux.NewRouter().StrictSlash(true),
		rcmd:   rcmd,
		logger: l,
	}
}

func (s *Service) services(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	log := s.logger.WithFields(logrus.Fields{
		"Component": "Router",
	})

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Error(err)
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
		return
	}

	rChan := make(chan []byte)

	s.rcmd.Push(rChan)
	reply := <-rChan
	if string(reply) != "0x0" {
		log.Errorf("rChan init was not 0x0")
		w.WriteHeader(500)
		w.Write([]byte("Internal server error"))
	}

	rChan <- body

rLoop:
	for {
		select {
		case reply := <-rChan:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			_, err = w.Write(reply)
			if err != nil {
				log.Error(err)
			}
			break rLoop
		case <-time.After(s.rcmd.DataTimeOut):
			w.Header().Set("Content-Type", "application/json")
			msg := fmt.Sprintf("Service channel did not respond within %d seconds. Closing connection", s.rcmd.DataTimeOut)
			log.Errorf(msg)
			w.WriteHeader(500)
			_, err = w.Write([]byte(msg))
			if err != nil {
				log.Error(err)
			}
			return
		}
	}

	for {
		select {
		case reply := <-rChan:
			if string(reply) != "0xF" {
				msg := "server error, expected 0xF but received " + string(reply)
				log.Errorf(msg)
			}
			return
		case <-time.After(10 * time.Second):
			msg := "Service channel did not respond within 10 seconds. Done waiting"
			log.Errorf(msg)
			return
		}
	}
}
