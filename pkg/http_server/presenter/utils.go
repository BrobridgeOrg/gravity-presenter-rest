package presenter

import (
	"strconv"
	"strings"
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
