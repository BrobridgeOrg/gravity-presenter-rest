package presenter

import (
	"encoding/json"
	"html/template"
	//"text/template"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
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
	//	Conditions map[string]string `json:"conditions"`
	Condition  *Condition `json:"condition"`
	Table      string     `json:"table"`
	Limit      int64      `json:"limit"`
	Offset     int64      `json:"offset"`
	OrderBy    string     `json:"orderBy"`
	Descending bool       `json:"descending"`
}

type ResponseConfig struct {
	ContentType string                     `json:"contentType"`
	State       map[string]StateDefinition `json:"state"`
}

type StateDefinition struct {
	ContentType string `json:"contentType"`
	Code        int    `json:"code"`
	Template    string `json:"template"`
	template    *template.Template
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

var defaultStates = map[string]StateDefinition{
	"success": StateDefinition{
		ContentType: "application/json",
		Code:        200,
	},
	"no_results": StateDefinition{
		ContentType: "application/json",
		Code:        404,
	},
}

type Param struct {
	pType    VariableType
	name     string
	source   string
	operator string
}

type Endpoint struct {
	presenter *Presenter
	name      string
	dirPath   string
	template  *template.Template
	method    string
	uri       string
	table     string
	params    map[string]Param
	response  *ResponseConfig
	states    map[string]*StateDefinition
	query     *QueryConfig
}

func NewEndpoint(presenter *Presenter, name string) *Endpoint {
	return &Endpoint{
		presenter: presenter,
		name:      name,
		params:    make(map[string]Param),
		states:    make(map[string]*StateDefinition),
	}
}

func (endpoint *Endpoint) loadCondition(queryConfig *QueryConfig, condition *Condition) error {

	if condition == nil {
		return nil
	}

	// Prepare value script
	if condition.Value != nil {
		condition.InitRuntime()
	}

	// Initializing child conditions
	for _, c := range condition.Conditions {
		return endpoint.loadCondition(queryConfig, c)
	}

	return nil
}

func (endpoint *Endpoint) loadQuerySettings(queryConfig *QueryConfig) error {

	endpoint.query = queryConfig

	return endpoint.loadCondition(queryConfig, queryConfig.Condition)
}

func (endpoint *Endpoint) Load(filename string) error {

	endpoint.dirPath = filepath.Dir(filename)

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

	// load condition settings
	err = endpoint.loadQuerySettings(&config.Query)
	if err != nil {
		return err
	}

	// Initialize response definitions
	err = endpoint.InitStates()
	if err != nil {
		return err
	}
	/*
		tmplFilename := strings.TrimSuffix(filename, filepath.Ext(filename)) + ".tmpl"
		err = endpoint.LoadTemplate(tmplFilename)
		if err != nil {
			return err
		}
	*/

	return nil
}

func (endpoint *Endpoint) InitStates() error {

	for stateName, defState := range defaultStates {

		state, ok := endpoint.response.State[stateName]
		if !ok {
			if s, ok := endpoint.response.State["success"]; ok {
				state = s
			} else {
				state = defState
			}
		} else {

			if state.Code == 0 {
				state.Code = 200
			}

			if len(state.ContentType) == 0 {
				state.ContentType = "application/json"
			}
		}

		tpName := ""
		if len(state.Template) == 0 {
			tpName = endpoint.name + ".tmpl"
			state.Template = filepath.Join(endpoint.dirPath, endpoint.name+".tmpl")
		} else if string(state.Template[0]) != "/" {
			tpName = state.Template
			state.Template = filepath.Join(endpoint.dirPath, state.Template)
		}

		// Load template
		tf := template.FuncMap{
			"counter": func(i int) int {
				return i + 1
			},
		}

		t, err := template.New(tpName).Funcs(tf).ParseFiles(state.Template)
		if err != nil {
			return err
		}

		state.template = t

		endpoint.states[stateName] = &state
	}

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

func (endpoint *Endpoint) prepareCondition(ctx *gin.Context, c *Condition) (*Condition, error) {

	if c == nil {
		if endpoint.query.Condition == nil {
			return nil, nil
		}

		c = endpoint.query.Condition
	}

	// Prepare a new condition which is based on template
	condition := &Condition{
		Name:       c.Name,
		Operator:   c.Operator,
		Conditions: make([]*Condition, 0, len(c.Conditions)),
	}

	condition.InitRuntime()

	// Prepare environment variable for script
	querys := make(map[string]string, len(ctx.Request.URL.Query()))
	for k, v := range ctx.Request.URL.Query() {
		querys[k] = v[0]
	}
	//condition.Runtime.Set("query", ctx.Request.URL.Query())
	condition.Runtime.Set("query", querys)

	// Path parameters
	params := make(map[string]interface{}, len(ctx.Params))
	//	params := condition.Runtime.NewObject()
	for _, p := range ctx.Params {
		//		params.Set(p.Key, p.Value)
		params[p.Key] = p.Value
	}

	//condition.Runtime.Set("param", mapper.NewParamObject(params))
	condition.Runtime.Set("param", params)

	// Body
	var body map[string]interface{}
	err := ctx.ShouldBind(&body)
	if err != nil {
		return nil, err
	}

	// Run script to get result
	if c.Value != nil {
		result, err := condition.Runtime.RunString(c.Value.(string))
		if err != nil {
			return nil, err
		}

		condition.Value = result.Export()
	}

	if c.Field != "" {
		result, err := condition.Runtime.RunString(c.Field)
		if err != nil {
			return nil, err
		} else {
			condition.Name = result.Export().(string)
		}
	}

	// Processing childs
	for _, child := range c.Conditions {
		sub, err := endpoint.prepareCondition(ctx, child)
		if err != nil {
			return nil, err
		}

		condition.Conditions = append(condition.Conditions, sub)
	}

	return condition, nil
}

func (endpoint *Endpoint) handler(c *gin.Context) {

	condition, err := endpoint.prepareCondition(c, nil)
	if err != nil {
		log.Error(err)
		c.Status(http.StatusBadRequest)
		c.Abort()
		return
	}
	/*
		conditions := make([]Condition, 0, len(endpoint.params))
		//	parameters := make(map[string]interface{})

		// Getting parameters
		for name, param := range endpoint.params {

			condition := Condition{
				Name:     name,
				Operator: param.operator,
			}

			switch param.pType {
			case VARIABLE_TYPE_QUERYSTRING:
				condition.Value = c.Query(param.source)
				//			parameters[name] = c.Query(param.source)
			case VARIABLE_TYPE_PARAMS:
				condition.Value = c.Param(param.source)
				//			parameters[name] = c.Param(param.source)
			case VARIABLE_TYPE_BODY:
				val := getValueFromObject(body, param.source)
				if val == nil {
					c.Status(http.StatusBadRequest)
					c.Abort()
				}

				//			parameters[name] = val
				condition.Value = val
			}

			conditions = append(conditions, condition)
		}
	*/
	// Query
	//	result, err := endpoint.presenter.queryAdapter.Query(endpoint.table, parameters, &QueryOption{})

	queryOption := QueryOption{
		Limit:      endpoint.query.Limit,
		Offset:     endpoint.query.Offset,
		OrderBy:    endpoint.query.OrderBy,
		Descending: endpoint.query.Descending,
	}

	result, err := endpoint.presenter.queryAdapter.Query(endpoint.table, condition, &queryOption)
	if err != nil {
		log.Error(err)
		c.Status(http.StatusInternalServerError)
		c.Abort()
		return
	}

	data := ViewData{
		Records: make([]map[string]interface{}, 0, len(result.Records)),
	}

	if len(result.Records) == 0 {

		// Render for no results
		state := endpoint.states["no_results"]
		c.Writer.Header().Set("Content-Type", state.ContentType)
		c.Status(state.Code)
		state.template.Execute(c.Writer, data)
		return
	}

	// Prepare records
	for _, record := range result.Records {

		row := make(map[string]interface{})

		for _, field := range record.Fields {
			row[field.Name] = GetValue(field.Value)
		}

		data.Records = append(data.Records, row)
	}

	// Render
	state := endpoint.states["success"]
	c.Writer.Header().Set("Content-Type", state.ContentType)
	state.template.Execute(c.Writer, data)
}
