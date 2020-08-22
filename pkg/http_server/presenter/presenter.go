package presenter

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/BrobridgeOrg/gravity-presenter-rest/pkg/http_server"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type Presenter struct {
	server    http_server.Server
	endpoints map[string]*Endpoint
}

func NewPresenter(server http_server.Server) *Presenter {
	return &Presenter{
		server:    server,
		endpoints: make(map[string]*Endpoint),
	}
}

func (presenter *Presenter) Init() error {

	settingsPath := viper.GetString("service.settingsPath")

	log.WithFields(log.Fields{
		"path": settingsPath,
	}).Info("Loading settings")

	err := filepath.Walk(settingsPath, func(path string, info os.FileInfo, err error) error {

		// Ignore directory
		if info.IsDir() {
			return nil
		}

		if filepath.Ext(path) != ".json" {
			return nil
		}

		log.WithFields(log.Fields{
			"filename": info.Name(),
		}).Info("Loading endpoint")

		// Create endpoint
		endpointName := strings.TrimSuffix(info.Name(), filepath.Ext(info.Name()))
		endpoint := NewEndpoint(endpointName)
		if err := endpoint.Load(path); err != nil {
			return err
		}

		if err := endpoint.Register(presenter.server); err != nil {
			return err
		}

		presenter.endpoints[endpointName] = endpoint

		return nil
	})

	if err != nil {
		return err
	}

	return nil
}
