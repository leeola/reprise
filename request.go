package reprise

import "encoding/json"

type Request struct {
	Step    int             `json:"-"`
	Method  string          `json:"method"`
	URLPath string          `json:"urlPath"`
	JSON    json.RawMessage `json:"json,omitempty"`
	Body    []byte          `json:"body,omitempty"`
}
