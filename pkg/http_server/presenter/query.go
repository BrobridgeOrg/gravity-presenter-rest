package presenter

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/gob"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/BrobridgeOrg/gravity-presenter-rest/pkg/http_server/presenter/pool"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"google.golang.org/grpc"

	querykit "github.com/BrobridgeOrg/gravity-api/service/querykit"
)

var (
	NotUnsignedIntegerErr = errors.New("Not unisgned integer")
	NotIntegerErr         = errors.New("Not integer")
	NotFloatErr           = errors.New("Not float")
)

type QueryOption struct {
	Limit      int64
	Offset     int64
	OrderBy    string
	Descending bool
}

type QueryAdapter struct {
	pool *pool.GRPCPool
}

func NewQueryAdapter() *QueryAdapter {
	return &QueryAdapter{}
}

func (adapter *QueryAdapter) Init() error {

	// Initialize connection pool
	host := fmt.Sprintf("%s:%d", viper.GetString("querykit.host"), viper.GetInt("querykit.port"))
	options := &pool.Options{
		InitCap:     8,
		MaxCap:      16,
		DialTimeout: time.Second * 20,
		IdleTimeout: time.Second * 60,
	}

	p, err := pool.NewGRPCPool(host, options, grpc.WithInsecure())
	if err != nil {
		return err
	}

	if p == nil {
		return err
	}

	adapter.pool = p

	return nil
}

func (adapter *QueryAdapter) prepareCondition(condition *Condition) (*querykit.Condition, error) {

	v, err := adapter.getValue(condition.Value)
	if err != nil {
		return nil, err
	}

	qCondition := &querykit.Condition{
		Name:       condition.Name,
		Value:      v,
		Conditions: make([]*querykit.Condition, len(condition.Conditions)),
	}

	if condition.Operator == ">" {
		qCondition.Operator = querykit.Operator_GREATER_THAN
	} else if condition.Operator == ">=" {
		qCondition.Operator = querykit.Operator_GREATER_EQUAL
	} else if condition.Operator == "<" {
		qCondition.Operator = querykit.Operator_LESS_THAN
	} else if condition.Operator == "<=" {
		qCondition.Operator = querykit.Operator_LESS_EQUAL
	} else {
		qCondition.Operator = querykit.Operator_EQUAL
	}

	/// Processing childs
	for _, child := range condition.Conditions {
		c, err := adapter.prepareCondition(child)
		if err != nil {
			return nil, err
		}

		qCondition.Conditions = append(qCondition.Conditions, c)
	}

	return qCondition, nil
}

//func (adapter *QueryAdapter) Query(table string, conditions map[string]interface{}, option *QueryOption) (*querykit.QueryReply, error) {
//func (adapter *QueryAdapter) Query(table string, conditions []Condition, option *QueryOption) (*querykit.QueryReply, error) {
func (adapter *QueryAdapter) Query(table string, condition *Condition, option *QueryOption) (*querykit.QueryReply, error) {

	conn, err := adapter.pool.Get()
	if err != nil {
		return nil, err
	}

	client := querykit.NewQueryKitClient(conn)
	adapter.pool.Put(conn)

	// Preparing request
	request := &querykit.QueryRequest{
		Table:      table,
		Limit:      option.Limit,
		Offset:     option.Offset,
		OrderBy:    option.OrderBy,
		Descending: option.Descending,
	}

	if condition != nil {
		qCondition, err := adapter.prepareCondition(condition)
		if err != nil {
			return nil, err
		}

		request.Condition = qCondition
	}

	/*
		//	for name, value := range conditions {
		for _, condition := range conditions {

			// Convert value to protobuf format
			//		v, err := adapter.getValue(value)
			v, err := adapter.getValue(condition.Value)
			if err != nil {
				log.Error(err)
				continue
			}

			//			request.Conditions = append(request.Conditions, &querykit.Condition{
			//				Name:  name,
			//				Value: v,
			//			})

			qCondition := &querykit.Condition{
				Name:  condition.Name,
				Value: v,
			}

			if condition.Operator == ">" {
				qCondition.Operator = querykit.Operator_GREATER_THAN
			} else if condition.Operator == ">=" {
				qCondition.Operator = querykit.Operator_GREATER_EQUAL
			} else if condition.Operator == "<" {
				qCondition.Operator = querykit.Operator_LESS_THAN
			} else if condition.Operator == "<=" {
				qCondition.Operator = querykit.Operator_LESS_EQUAL
			} else {
				qCondition.Operator = querykit.Operator_EQUAL
			}

			request.Conditions = append(request.Conditions, qCondition)

		}
	*/

	reply, err := client.Query(context.Background(), request)
	if err != nil {
		return nil, err
	}

	return reply, nil
}

func (adapter *QueryAdapter) getValue(data interface{}) (*querykit.Value, error) {

	if data == nil {
		return nil, errors.New("data cannnot be nil")
	}

	// Float
	bytes, err := adapter.getBytesFromFloat(data)
	if err == nil {
		return &querykit.Value{
			Type:  querykit.DataType_FLOAT64,
			Value: bytes,
		}, nil
	}

	// Integer
	bytes, err = adapter.getBytesFromInteger(data)
	if err == nil {
		return &querykit.Value{
			Type:  querykit.DataType_INT64,
			Value: bytes,
		}, nil
	}

	// Unsigned integer
	bytes, err = adapter.getBytesFromUnsignedInteger(data)
	if err == nil {
		return &querykit.Value{
			Type:  querykit.DataType_INT64,
			Value: bytes,
		}, nil
	}

	v := reflect.ValueOf(data)

	switch v.Kind() {
	case reflect.Bool:
		data, _ := adapter.getBytes(data)
		return &querykit.Value{
			Type:  querykit.DataType_BOOLEAN,
			Value: data,
		}, nil
	case reflect.String:
		return &querykit.Value{
			Type:  querykit.DataType_STRING,
			Value: []byte(data.(string)),
		}, nil
	case reflect.Map:

		// Prepare map value
		value := querykit.MapValue{
			Fields: make([]*querykit.Field, 0),
		}

		// Convert each key-value set
		for _, key := range v.MapKeys() {
			ele := v.MapIndex(key)

			// Convert value to protobuf format
			v, err := adapter.getValue(ele.Interface())
			if err != nil {
				log.Error(err)
				continue
			}

			field := querykit.Field{
				Name:  key.Interface().(string),
				Value: v,
			}

			value.Fields = append(value.Fields, &field)
		}

		return &querykit.Value{
			Type: querykit.DataType_MAP,
			Map:  &value,
		}, nil

	case reflect.Slice:

		// Prepare map value
		value := querykit.ArrayValue{
			Elements: make([]*querykit.Value, 0, v.Len()),
		}

		for i := 0; i < v.Len(); i++ {
			ele := v.Index(i)

			// Convert value to protobuf format
			v, err := adapter.getValue(ele.Interface())
			if err != nil {
				log.Error(err)
				continue
			}

			value.Elements = append(value.Elements, v)
		}

		return &querykit.Value{
			Type:  querykit.DataType_ARRAY,
			Array: &value,
		}, nil

	default:
		data, _ := adapter.getBytes(data)
		return &querykit.Value{
			Type:  querykit.DataType_BINARY,
			Value: data,
		}, nil
	}
}

func (adapter *QueryAdapter) getBytes(data interface{}) ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(data)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (adapter *QueryAdapter) getBytesFromUnsignedInteger(data interface{}) ([]byte, error) {

	var buf = make([]byte, 8)

	v := reflect.ValueOf(data)
	switch v.Kind() {
	case reflect.Uint:
		binary.LittleEndian.PutUint64(buf, uint64(data.(uint)))
	case reflect.Uint8:
		binary.LittleEndian.PutUint64(buf, uint64(data.(uint8)))
	case reflect.Uint16:
		binary.LittleEndian.PutUint64(buf, uint64(data.(uint16)))
	case reflect.Uint32:
		binary.LittleEndian.PutUint64(buf, uint64(data.(uint32)))
	case reflect.Uint64:
		binary.LittleEndian.PutUint64(buf, data.(uint64))
	default:
		return nil, NotUnsignedIntegerErr
	}

	return buf, nil
}

func (adapter *QueryAdapter) getBytesFromInteger(data interface{}) ([]byte, error) {

	var buf = make([]byte, 8)

	v := reflect.ValueOf(data)
	switch v.Kind() {
	case reflect.Int:
		binary.LittleEndian.PutUint64(buf, uint64(data.(int)))
	case reflect.Int8:
		binary.LittleEndian.PutUint64(buf, uint64(data.(int8)))
	case reflect.Int16:
		binary.LittleEndian.PutUint64(buf, uint64(data.(int16)))
	case reflect.Int32:
		binary.LittleEndian.PutUint64(buf, uint64(data.(int32)))
	case reflect.Int64:
		binary.LittleEndian.PutUint64(buf, uint64(data.(int64)))
	default:
		return nil, NotIntegerErr
	}

	return buf, nil
}

func (adapter *QueryAdapter) getBytesFromFloat(data interface{}) ([]byte, error) {
	var buf = make([]byte, 8)

	v := reflect.ValueOf(data)
	switch v.Kind() {
	case reflect.Float32:
		binary.LittleEndian.PutUint64(buf, uint64(data.(float32)))
	case reflect.Float64:
		binary.LittleEndian.PutUint64(buf, uint64(data.(float64)))
	default:
		return nil, NotFloatErr
	}

	return buf, nil
}
