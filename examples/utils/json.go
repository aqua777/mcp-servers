package utils

import "encoding/json"

// JsonStr marshals a value into a pretty JSON string.
func JsonStr(v interface{}) string {
	b, _ := json.MarshalIndent(v, "", "  ")
	return string(b)
}
