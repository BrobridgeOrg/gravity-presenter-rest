package server

import (
	"net"
	"net/http"

	app "github.com/BrobridgeOrg/gravity-presenter-rest/pkg/app"
	presenter "github.com/BrobridgeOrg/gravity-presenter-rest/pkg/http_server/presenter"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"github.com/soheilhy/cmux"
)

type Server struct {
	app      app.App
	engine   *gin.Engine
	instance *http.Server
	listener net.Listener
	host     string

	presenter *presenter.Presenter
}

func NewServer(a app.App) *Server {
	return &Server{
		app:      a,
		instance: &http.Server{},
	}
}

func (server *Server) Init(host string) error {

	// Put it to mux
	mux, err := server.app.GetMuxManager().AssertMux("http", host)
	if err != nil {
		return err
	}

	// Preparing listener
	lis := mux.Match(cmux.HTTP1Fast())
	server.host = host
	server.listener = lis
	server.engine = gin.Default()

	// APIs
	//	api.NewAuth(server.app, server).Register()

	// Initializing presenter
	server.presenter = presenter.NewPresenter(server)
	err = server.presenter.Init()
	if err != nil {
		return err
	}

	server.instance.Handler = server.engine

	return nil
}

func (server *Server) Serve() error {

	log.WithFields(log.Fields{
		"host": server.host,
	}).Info("Starting HTTP server")

	// Starting server
	if err := server.instance.Serve(server.listener); err != cmux.ErrListenerClosed {
		log.Error(err)
		return err
	}

	return nil
}

func (server *Server) GetEngine() *gin.Engine {
	return server.engine
}

func (server *Server) GetApp() app.App {
	return server.app
}
