package tools

import (
	"fmt"
	"strconv"
)

// GetStringParam safely gets a string parameter from arguments
// It also handles numeric IDs and converts them to strings
func GetStringParam(arguments map[string]interface{}, key string, required bool) (string, error) {
	val, ok := arguments[key]
	if !ok {
		if required {
			return "", fmt.Errorf("missing required argument: %s", key)
		}
		return "", nil
	}

	switch v := val.(type) {
	case string:
		return v, nil
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64), nil
	case int:
		return strconv.Itoa(v), nil
	case int64:
		return strconv.FormatInt(v, 10), nil
	default:
		return "", fmt.Errorf("invalid type for argument %s: expected string or number, got %T", key, val)
	}
}

// GetObjectParam safely gets a map/object parameter from arguments
func GetObjectParam(arguments map[string]interface{}, key string, required bool) (map[string]interface{}, error) {
	val, ok := arguments[key]
	if !ok {
		if required {
			return nil, fmt.Errorf("missing required argument: %s", key)
		}
		return nil, nil
	}

	obj, ok := val.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid type for argument %s: expected object", key)
	}

	return obj, nil
}

// GetIntParam safely gets an integer parameter from arguments
func GetIntParam(arguments map[string]interface{}, key string, required bool) (int, error) {
	val, ok := arguments[key]
	if !ok {
		if required {
			return 0, fmt.Errorf("missing required argument: %s", key)
		}
		return 0, nil
	}

	switch v := val.(type) {
	case float64:
		return int(v), nil
	case int:
		return v, nil
	case int64:
		return int(v), nil
	case string:
		return strconv.Atoi(v)
	default:
		return 0, fmt.Errorf("invalid type for argument %s: expected number or string, got %T", key, val)
	}
}

// GetBoolParam safely gets a boolean parameter from arguments
func GetBoolParam(arguments map[string]interface{}, key string, required bool) (bool, error) {
	val, ok := arguments[key]
	if !ok {
		if required {
			return false, fmt.Errorf("missing required argument: %s", key)
		}
		return false, nil
	}

	switch v := val.(type) {
	case bool:
		return v, nil
	case string:
		return strconv.ParseBool(v)
	default:
		return false, fmt.Errorf("invalid type for argument %s: expected boolean or string, got %T", key, val)
	}
}

// GetPaginationParams extracts pagination parameters (limit, cursor)
func GetPaginationParams(arguments map[string]interface{}) (map[string]interface{}, error) {
	params := make(map[string]interface{})

	if limit, ok := arguments["limit"]; ok {
		params["limit"] = limit
	}

	if cursor, ok := arguments["cursor"]; ok {
		params["cursor"] = cursor
	}

	return params, nil
}

// AddPaginationToQuery adds pagination parameters to query map
func AddPaginationToQuery(query map[string]string, pagination map[string]interface{}) {
	if limit, ok := pagination["limit"]; ok {
		switch v := limit.(type) {
		case float64:
			query["limit"] = strconv.FormatFloat(v, 'f', -1, 64)
		case int:
			query["limit"] = strconv.Itoa(v)
		}
	}

	if cursor, ok := pagination["cursor"]; ok {
		if s, ok := cursor.(string); ok {
			query["cursor"] = s
		}
	}
}

// GetArrayParam safely gets an array parameter from arguments
func GetArrayParam(arguments map[string]interface{}, key string, required bool) ([]interface{}, error) {
	val, ok := arguments[key]
	if !ok {
		if required {
			return nil, fmt.Errorf("missing required argument: %s", key)
		}
		return nil, nil
	}

	arr, ok := val.([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid type for argument %s: expected array", key)
	}

	return arr, nil
}

// GetStringArrayParam safely gets a string array parameter from arguments
func GetStringArrayParam(arguments map[string]interface{}, key string, required bool) ([]string, error) {
	arr, err := GetArrayParam(arguments, key, required)
	if err != nil {
		return nil, err
	}
	if arr == nil {
		return nil, nil
	}

	result := make([]string, 0, len(arr))
	for i, v := range arr {
		s, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("invalid type for element %d of argument %s: expected string", i, key)
		}
		result = append(result, s)
	}

	return result, nil
}
