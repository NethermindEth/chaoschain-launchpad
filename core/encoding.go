package core

import "encoding/json"

func DecodeJSON(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

func EncodeJSON(v interface{}) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	return data
}
