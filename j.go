package ap

import "strconv"

type J map[string]any

func NewJ(v interface{}) J {
	return J(v.(map[string]interface{}))
}

func (j J) Get(key string) string {
	if v, ok := j[key].(string); ok {
		return v
	}
	if v, ok := j[key].(float64); ok {
		return strconv.FormatFloat(v, 'f', 0, 64)
	}
	return ""
}

func (j J) GetI(key string) int64 {
	if v, ok := j[key].(int64); ok {
		return v
	}
	if v, ok := j[key].(float64); ok {
		return int64(v)
	}
	if v, ok := j[key].(string); ok {
		i, err := strconv.Atoi(v)
		if err != nil {
			panic(err)
		}
		return int64(i)
	}
	return 0
}

func (j J) GetB(key string) bool {
	if v, ok := j[key].(bool); ok {
		return v
	}
	if v, ok := j[key].(string); ok {
		return v == "true"
	}
	return false
}

func (j J) GetJ(key string) J {
	return NewJ(j[key])
}

func (j J) Set(key string, value interface{}) string {
	j[key] = value
	return "" // return something that wont display do this can be used in templates
}
