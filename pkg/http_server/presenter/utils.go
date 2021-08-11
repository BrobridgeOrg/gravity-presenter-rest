package presenter

import (
	"encoding/binary"
	"math"
	"strconv"
	"strings"

	querykit "github.com/BrobridgeOrg/gravity-api/service/querykit"
)

func getValueFromObject(obj interface{}, targetPath string) interface{} {

	parts := strings.Split(targetPath, ".")

	cursor := obj
	for _, name := range parts {

		switch cursor.(type) {
		case map[string]interface{}:
			val, ok := cursor.(map[string]interface{})[name]
			if !ok {
				return nil
			}

			cursor = val
		case []interface{}:

			index, err := strconv.ParseInt(name, 10, 64)
			if err != nil {
				return nil
			}

			arr := cursor.([]interface{})
			if len(arr) > int(index) {
				return arr[index]
			}

			return nil
		}
	}

	return cursor
}

func GetValue(value *querykit.Value) interface{} {

	if value == nil {
		return nil
	}

	switch value.Type {
	case querykit.DataType_FLOAT64:
		return math.Float64frombits(binary.LittleEndian.Uint64(value.Value))
	case querykit.DataType_INT64:
		return int64(binary.LittleEndian.Uint64(value.Value))
	case querykit.DataType_UINT64:
		return uint64(binary.LittleEndian.Uint64(value.Value))
	case querykit.DataType_BOOLEAN:
		return int8(value.Value[0]) & 1
	case querykit.DataType_STRING:
		return string(value.Value)
	}

	// binary
	return value.Value
}
