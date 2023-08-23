package common

import "errors"

func AnyToMap(a any) (map[string]string, error) {
	// Check if the input is of type map[string]string
	if myMap, ok := a.(map[string]string); ok {
		return myMap, nil
	}

	// Check if the input is of type map[string]interface{}
	if myMap, ok := a.(map[string]interface{}); ok {
		// Convert map[string]interface{} to map[string]string
		result := make(map[string]string)
		for key, val := range myMap {
			strVal, oks := val.(string)
			if !oks {
				return nil, errors.New("map value is not a string")
			}
			result[key] = strVal
		}
		return result, nil
	}

	return nil, errors.New("interface is not a map")
}
