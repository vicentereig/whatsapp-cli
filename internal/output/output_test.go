package output

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSuccess(t *testing.T) {
	tests := []struct {
		name string
		data interface{}
		want string
	}{
		{
			name: "simple string",
			data: "hello",
			want: `{"success":true,"data":"hello","error":null}`,
		},
		{
			name: "struct data",
			data: map[string]string{"name": "John"},
			want: `{"success":true,"data":{"name":"John"},"error":null}`,
		},
		{
			name: "nil data",
			data: nil,
			want: `{"success":true,"data":null,"error":null}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Success(tt.data)
			assert.JSONEq(t, tt.want, got)
		})
	}
}

func TestError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "simple error",
			err:  assert.AnError,
			want: `{"success":false,"data":null,"error":"assert.AnError general error for testing"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Error(tt.err)
			assert.JSONEq(t, tt.want, got)
		})
	}
}

func TestResult_JSON(t *testing.T) {
	r := Result{
		Success: true,
		Data:    []string{"a", "b"},
		Error:   nil,
	}

	got, err := json.Marshal(r)
	assert.NoError(t, err)
	assert.JSONEq(t, `{"success":true,"data":["a","b"],"error":null}`, string(got))
}
