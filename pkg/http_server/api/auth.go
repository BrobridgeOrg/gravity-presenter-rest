package api

import (
	"log"
	"net/http"

	"git.brobridge.com/pilotwave/pilotwave/pkg/app"
	"git.brobridge.com/pilotwave/pilotwave/pkg/auth"
	"git.brobridge.com/pilotwave/pilotwave/pkg/http_server"
	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	validation "github.com/go-ozzo/ozzo-validation"
	"github.com/spf13/viper"
)

type Auth struct {
	app    app.App
	server http_server.Server
	router *gin.RouterGroup
}

type SignInRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func NewAuth(a app.App, s http_server.Server) *Auth {
	return &Auth{
		app:    a,
		server: s,
	}
}

func (api *Auth) Register() {

	api.router = api.server.GetEngine().Group("/api/v1/auth")
	//	g.Router.Use(middlewares.RequiredAuth())

	api.router.POST("/signin", api.SignIn)

	//	r := server.GetEngine()
}

func (api *Auth) SignIn(c *gin.Context) {

	// Parsing body
	var body SignInRequest
	err := c.BindJSON(&body)
	if err != nil {
		log.Println(err)
		c.Status(http.StatusBadRequest)
		c.Abort()
		return
	}

	// Validate fields
	err = validation.ValidateStruct(&body,
		validation.Field(&body.Username, validation.Required),
		validation.Field(&body.Password, validation.Required),
	)
	if err != nil {
		// TODO: return 409 and error messages
		log.Println(err)
		c.Status(http.StatusBadRequest)
		c.Abort()
		return
	}

	authenticator := api.app.GetAuthenticator()

	// Authenticate with username and password
	var resp *auth.AuthenticateResponse
	if viper.GetString("auth.method") != "ad" {

		// Built-in
		resp, err = authenticator.Authenticate(body.Username, body.Password)
		if err != nil {
			c.Status(http.StatusInternalServerError)
			c.Abort()
			return
		}
	} else {

		// Active directory or LDAP
		resp, err = authenticator.AuthenticateWithAD(body.Username, body.Password)
		if err != nil {
			c.Status(http.StatusInternalServerError)
			c.Abort()
			return
		}
	}

	if resp == nil {
		c.Status(http.StatusUnauthorized)
		c.Abort()
		return
	}

	// TODO: check permission for logging

	// Build a token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"uid":      resp.ID,
		"username": resp.Username,
	})

	secret := viper.GetString("auth.secret")

	// Sign and get the complete encoded token as a string using the secret
	tokenStr, _ := token.SignedString([]byte(secret))

	// Response
	c.JSON(http.StatusOK, gin.H{
		"uid":         resp.ID,
		"name":        resp.Name,
		"username":    resp.Username,
		"permissions": resp.Name,
		"token":       tokenStr,
	})
}
