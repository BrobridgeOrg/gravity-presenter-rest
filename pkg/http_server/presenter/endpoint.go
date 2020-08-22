package presenter

import (
	"encoding/json"
	"errors"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/BrobridgeOrg/gravity-presenter-rest/pkg/http_server"
	"github.com/gin-gonic/gin"
)

type EndpointConfig struct {
	Method string `json:"method"`
	Uri    string `json:"uri"`
	Query  map[string]string
}

type VariableType int

const (
	VARIABLE_TYPE_PARAMS VariableType = iota
	VARIABLE_TYPE_QUERYSTRING
	VARIABLE_TYPE_BODY
)

var varTypes map[string]VariableType = map[string]VariableType{
	"param":       VARIABLE_TYPE_PARAMS,
	"querystring": VARIABLE_TYPE_QUERYSTRING,
	"body":        VARIABLE_TYPE_BODY,
}

type Param struct {
	pType  VariableType
	name   string
	source string
}

type Endpoint struct {
	name     string
	template *template.Template
	method   string
	uri      string
	params   map[string]Param
}

func NewEndpoint(name string) *Endpoint {
	return &Endpoint{
		name:   name,
		params: make(map[string]Param),
	}
}

func (endpoint *Endpoint) Load(filename string) error {

	// Open and read file
	jsonFile, err := os.Open(filename)
	if err != nil {
		return err
	}

	defer jsonFile.Close()

	byteValue, _ := ioutil.ReadAll(jsonFile)

	// Parse JSON
	var config EndpointConfig
	err = json.Unmarshal(byteValue, &config)
	if err != nil {
		return err
	}

	endpoint.method = config.Method
	endpoint.uri = config.Uri

	for name, def := range config.Query {

		// Split with dot
		re := regexp.MustCompile(`\.`)
		parts := re.Split(def, 2)

		if len(parts) != 2 {
			return errors.New("Uncoganized query settings")
		}

		// First part is parameter type
		pType, ok := varTypes[parts[0]]
		if !ok {
			return errors.New("No such parameter type for query")
		}

		param := Param{
			pType:  pType,
			name:   name,
			source: parts[1],
		}

		endpoint.params[param.name] = param
	}

	// Load template
	tmplFilename := strings.TrimSuffix(filename, filepath.Ext(filename)) + ".tmpl"
	err = endpoint.LoadTemplate(tmplFilename)
	if err != nil {
		return err
	}

	return nil
}

func (endpoint *Endpoint) LoadTemplate(filename string) error {

	t, err := template.ParseFiles(filename)
	if err != nil {
		return err
	}

	endpoint.template = t

	return nil
}

func (endpoint *Endpoint) Register(server http_server.Server) error {

	switch endpoint.method {
	case "post":
		server.GetEngine().POST(endpoint.uri, endpoint.handler)
	case "get":
		server.GetEngine().GET(endpoint.uri, endpoint.handler)
	case "delete":
		server.GetEngine().DELETE(endpoint.uri, endpoint.handler)
	case "put":
		server.GetEngine().PUT(endpoint.uri, endpoint.handler)
	}

	return nil
}

func (endpoint *Endpoint) handler(c *gin.Context) {

	// Parsing body
	var body map[string]interface{}
	err := c.ShouldBind(&body)
	if err != nil {
		c.Status(http.StatusBadRequest)
		c.Abort()
		return
	}

	var parameters map[string]interface{}

	// Getting parameters
	for name, param := range endpoint.params {

		switch param.pType {
		case VARIABLE_TYPE_QUERYSTRING:
			parameters[name] = c.Query(param.source)
		case VARIABLE_TYPE_PARAMS:
			parameters[name] = c.Param(param.source)
		case VARIABLE_TYPE_BODY:
			val := getValueFromObject(body, param.source)
			if val == nil {
				c.Status(http.StatusBadRequest)
				c.Abort()
			}

			parameters[name] = val
		}
	}

	data := map[string]interface{}{
		"accountType": "01",
		"accountName": "TEST",
	}

	endpoint.template.Execute(c.Writer, data)
}
