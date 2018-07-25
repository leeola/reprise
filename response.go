package reprise

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/middleware"
)

type Response struct {
	JSON  json.RawMessage `json:"json"`
	Bytes []byte          `json:"bytes"`
}

type ResponseTee struct {
	http.ResponseWriter
	tee bytes.Buffer
}

func NewResponseTee(w http.ResponseWriter, r *http.Request) (*ResponseTee, error) {
	rr := &ResponseTee{}

	// using chi to implement the Tee functionality. Why reinvent the wheel?
	ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
	ww.Tee(&rr.tee)
	rr.ResponseWriter = ww

	return rr, nil
}

func (rt *ResponseTee) Response() (Response, error) {
	res := Response{}

	if rt.tee.Len() != 0 {
		// don't think Bytes() is intended to be used this way,
		// perhaps it would be better to read the buffer, ensuring
		// no funny business goes on such as modifications or w/e.
		b := rt.tee.Bytes()

		var indentedJSON bytes.Buffer
		if err := json.Indent(&indentedJSON, b, "", "  "); err != nil {
			res.Bytes = b
		} else {
			res.JSON = indentedJSON.Bytes()
		}
	}

	return res, nil
}
