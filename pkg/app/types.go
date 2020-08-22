package app

import (
	"github.com/BrobridgeOrg/gravity-presenter-rest/pkg/http_server"
	"github.com/BrobridgeOrg/gravity-presenter-rest/pkg/mux_manager"
)

type App interface {
	GetMuxManager() mux_manager.Manager
	GetHTTPServer() http_server.Server
}
