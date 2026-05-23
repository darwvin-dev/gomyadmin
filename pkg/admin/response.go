package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
)

type Response struct {
	Data  any            `json:"data"`
	Meta  map[string]any `json:"meta"`
	Error *Error         `json:"error"`
}

type Error struct {
	Code    string              `json:"code"`
	Message string              `json:"message"`
	Fields  map[string][]string `json:"fields,omitempty"`
}

func WriteJSON(w http.ResponseWriter, status int, requestID string, data any, meta map[string]any) {
	if meta == nil {
		meta = map[string]any{}
	}
	if requestID != "" {
		meta["request_id"] = requestID
	}

	write(w, status, Response{Data: data, Meta: meta})
}

func WriteError(w http.ResponseWriter, status int, requestID, code, message string, fields map[string][]string) {
	meta := map[string]any{}
	if requestID != "" {
		meta["request_id"] = requestID
	}
	write(w, status, Response{
		Data: nil,
		Meta: meta,
		Error: &Error{
			Code:    code,
			Message: message,
			Fields:  fields,
		},
	})
}

func write(w http.ResponseWriter, status int, response Response) {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(response); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write(buf.Bytes())
}
