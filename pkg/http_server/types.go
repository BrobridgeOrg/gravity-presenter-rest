package http_server

import (
	"github.com/gin-gonic/gin"
)

type Server interface {
	Init(string) error
	Serve() error
	GetEngine() *gin.Engine
}
