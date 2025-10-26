package output

import "encoding/json"

type Result struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
	Error   *string     `json:"error"`
}

func Success(data interface{}) string {
	r := Result{
		Success: true,
		Data:    data,
		Error:   nil,
	}
	b, _ := json.Marshal(r)
	return string(b)
}

func Error(err error) string {
	errMsg := err.Error()
	r := Result{
		Success: false,
		Data:    nil,
		Error:   &errMsg,
	}
	b, _ := json.Marshal(r)
	return string(b)
}
