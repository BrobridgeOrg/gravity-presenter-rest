package instance

import (
	http_server "github.com/BrobridgeOrg/gravity-presenter-rest/pkg/http_server/server"
	mux_manager "github.com/BrobridgeOrg/gravity-presenter-rest/pkg/mux_manager/manager"
	log "github.com/sirupsen/logrus"
)

type AppInstance struct {
	done       chan bool
	muxManager *mux_manager.MuxManager
	httpServer *http_server.Server
}

func NewAppInstance() *AppInstance {

	a := &AppInstance{
		done: make(chan bool),
	}

	return a
}

func (a *AppInstance) Init() error {

	log.Info("Starting application")

	// Initializing modules
	a.muxManager = mux_manager.NewMuxManager(a)
	a.httpServer = http_server.NewServer(a)

	a.initMuxManager()

	// Initializing HTTP server
	err := a.initHTTPServer()
	if err != nil {
		return err
	}

	return nil
}

func (a *AppInstance) Uninit() {
}

func (a *AppInstance) Run() error {

	// HTTP
	go func() {
		err := a.runHTTPServer()
		if err != nil {
			log.Error(err)
		}
	}()

	err := a.runMuxManager()
	if err != nil {
		return err
	}

	<-a.done

	return nil
}
