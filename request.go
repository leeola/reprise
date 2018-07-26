package reprise

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
)

type Request struct {
	Step       int             `json:"-"`
	Method     string          `json:"method"`
	URI        string          `json:"uri"`
	BodyJSON   json.RawMessage `json:"bodyJson,omitempty"`
	BodyBinary []byte          `json:"bodyBinary,omitempty"`
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
			r.BodyBinary = b
		} else {
			r.BodyJSON = buf.Bytes()
		}
	}

	return r, nil
}

// BodyBytes returns the bytes for this request body, if any.
func (r *Request) BodyBytes() []byte {
	if r.BodyJSON != nil {
		return r.BodyJSON
	}
	return r.BodyBinary
}

// Body returns a reader for this request body, if any.
func (r *Request) Body() io.Reader {
	if b := r.BodyBytes(); b != nil {
		return bytes.NewReader(b)
	}
	return nil
}

func (r *Request) HTTPRequest(urlStr string) (*http.Request, error) {
	u, err := url.Parse(urlStr)
	if err != nil {
		return nil, fmt.Errorf("url parse: %v", err)
	}

	uri, err := u.Parse(r.URI)
	if err != nil {
		return nil, fmt.Errorf("uri parse: %v", err)
	}

	u.Path = path.Join(u.Path, uri.Path)

	// merge the two queries and fragments. A bit of an odd feature,
	// but seems unlikely to do harm, and only be beneficial.
	q := u.Query()
	for k, v := range uri.Query() {
		q[k] = v
	}
	u.RawQuery = q.Encode()
	if uri.Fragment != "" {
		u.Fragment = uri.Fragment
	}

	req, err := http.NewRequest(r.Method, u.String(), r.Body())
	if err != nil {
		return nil, fmt.Errorf("http.newrequest: %v", err)
	}

	return req, nil
}
