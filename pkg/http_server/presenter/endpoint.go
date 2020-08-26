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

	"github.com/gin-gonic/gin"
	"github.com/prometheus/common/log"
)

type ViewData struct {
	Records []map[string]interface{}
}

type EndpointConfig struct {
	Method   string         `json:"method"`
	Uri      string         `json:"uri"`
	Query    QueryConfig    `json:"query"`
	Response ResponseConfig `json:"response"`
}

type QueryConfig struct {
	Conditions map[string]string `json:"conditions"`
	Table      string            `json:"table"`
	Limit      int64             `json:"limit"`
	Offset     int64             `json:"offset"`
	OrderBy    string            `json:"orderBy"`
	Descending bool              `json:"descending"`
}

type ResponseConfig struct {
	ContentType string `json:"contentType"`
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
	presenter *Presenter
	name      string
	template  *template.Template
	method    string
	uri       string
	table     string
	params    map[string]Param
	response  *ResponseConfig
}

func NewEndpoint(presenter *Presenter, name string) *Endpoint {
	return &Endpoint{
		presenter: presenter,
		name:      name,
		params:    make(map[string]Param),
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
	endpoint.table = config.Query.Table
	endpoint.response = &config.Response

	if len(endpoint.response.ContentType) == 0 {
		endpoint.response.ContentType = "application/json"
	}

	for name, def := range config.Query.Conditions {

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

func (endpoint *Endpoint) Register() error {

	switch endpoint.method {
	case "post":
		endpoint.presenter.server.GetEngine().POST(endpoint.uri, endpoint.handler)
	case "get":
		endpoint.presenter.server.GetEngine().GET(endpoint.uri, endpoint.handler)
	case "delete":
		endpoint.presenter.server.GetEngine().DELETE(endpoint.uri, endpoint.handler)
	case "put":
		endpoint.presenter.server.GetEngine().PUT(endpoint.uri, endpoint.handler)
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

	parameters := make(map[string]interface{})

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

	// Query
	result, err := endpoint.presenter.queryAdapter.Query(endpoint.table, parameters, &QueryOption{})
	if err != nil {
		log.Error(err)
		c.Status(http.StatusInternalServerError)
		c.Abort()
		return
	}

	if len(result.Records) == 0 {
		// TODO
	}

	data := ViewData{
		Records: make([]map[string]interface{}, 0),
	}

	for _, record := range result.Records {

		row := make(map[string]interface{})

		for _, field := range record.Fields {
			row[field.Name] = GetValue(field.Value)
		}

		data.Records = append(data.Records, row)
	}
	/*
		record := map[string]interface{}{
			"accountType": "01",
			"accountName": "TEST",
		}

		data.Records = append(data.Records, record)
	*/
	c.Writer.Header().Set("Content-Type", endpoint.response.ContentType)

	endpoint.template.Execute(c.Writer, data)
}
