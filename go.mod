module github.com/BrobridgeOrg/gravity-presenter-rest

go 1.13

require (
	github.com/BrobridgeOrg/gravity-api v0.2.8
	github.com/BrobridgeOrg/gravity-exporter-rest v0.0.0-20200808213905-40fa5031150c
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/dlclark/regexp2 v1.4.0 // indirect
	github.com/dop251/goja v0.0.0-20201221183957-6b6d5e2b5d80
	github.com/gin-gonic/gin v1.6.3
	github.com/go-ozzo/ozzo-validation v3.6.0+incompatible
	github.com/go-sourcemap/sourcemap v2.1.3+incompatible // indirect
	github.com/jinzhu/gorm v1.9.16 // indirect
	github.com/sirupsen/logrus v1.6.0
	github.com/soheilhy/cmux v0.1.4
	github.com/spf13/viper v1.7.1
	golang.org/x/net v0.0.0-20200324143707-d3edc9973b7e
	google.golang.org/grpc v1.31.0
)

//replace github.com/BrobridgeOrg/gravity-api => ../gravity-api
