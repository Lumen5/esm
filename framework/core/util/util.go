package util

import "encoding/json"

// SubString safely returns a substring from start to end position
func SubString(s string, start, end int) string {
	if start < 0 {
		start = 0
	}
	if start >= len(s) {
		return ""
	}
	if end > len(s) {
		end = len(s)
	}
	if end <= start {
		return ""
	}
	return s[start:end]
}

// ToJson converts an object to JSON string
func ToJson(v interface{}, indent bool) string {
	var data []byte
	var err error
	if indent {
		data, err = json.MarshalIndent(v, "", "  ")
	} else {
		data, err = json.Marshal(v)
	}
	if err != nil {
		return ""
	}
	return string(data)
}

// ToJSONBytes converts an object to JSON bytes
func ToJSONBytes(v interface{}) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	return data
}
