package reprise

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

type Request struct {
	Step   int             `json:"-"`
	Method string          `json:"method"`
	URI    string          `json:"uri"`
	JSON   json.RawMessage `json:"json,omitempty"`
	Bytes  []byte          `json:"bytes,omitempty"`
}

func NewRequest(httpReq *http.Request) (Request, error) {
	r := Request{
		Method: httpReq.Method,
		URI:    httpReq.URL.RequestURI(),
	}

	if httpReq.Body != nil {
		b, err := ioutil.ReadAll(httpReq.Body)
		if err != nil {
			return Request{}, fmt.Errorf("readall: %v", err)
		}

		var buf bytes.Buffer
		if err := json.Indent(&buf, b, "", "  "); err != nil {
			r.Bytes = b
		} else {
			r.JSON = buf.Bytes()
		}
	}

	return r, nil
}
