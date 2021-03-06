package reprise

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/go-chi/chi/middleware"
)

type Response struct {
	BodyJSON   json.RawMessage `json:"bodyJson,omitempty"`
	BodyBinary []byte          `json:"bodyBinary,omitempty"`
}

type ResponseWriterTee struct {
	http.ResponseWriter
	tee bytes.Buffer
}

func NewResponseWriterTee(w http.ResponseWriter, r *http.Request) (*ResponseWriterTee, error) {
	rr := &ResponseWriterTee{}

	// using chi to implement the Tee functionality. Why reinvent the wheel?
	ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
	ww.Tee(&rr.tee)
	rr.ResponseWriter = ww

	return rr, nil
}

func NewResponse(r *http.Response) (Response, error) {
	res := Response{}

	if r.Body != nil {
		defer r.Body.Close()

		jsonB, bytesB, err := readerBytesOrJSON(r.Body)
		if err != nil {
			return Response{}, err // no wrap
		}

		res.BodyBinary = bytesB
		res.BodyJSON = jsonB
	}

	return res, nil
}

func (r Response) IsJSON() bool {
	return r.BodyJSON != nil
}

// BodyBytes returns the bytes for this request body, if any.
func (r *Response) BodyBytes() []byte {
	if r.BodyJSON != nil {
		return r.BodyJSON
	}
	return r.BodyBinary
}

// Body returns a reader for this request body, if any.
func (r *Response) Body() io.Reader {
	if b := r.BodyBytes(); b != nil {
		return bytes.NewReader(b)
	}
	return nil
}

func (rt *ResponseWriterTee) Response() (Response, error) {
	res := Response{}

	jsonB, bytesB, err := readerBytesOrJSON(&rt.tee)
	if err != nil {
		return Response{}, err // no wrap
	}

	res.BodyBinary = bytesB
	res.BodyJSON = jsonB

	return res, nil
}

func readerBytesOrJSON(r io.Reader) (jsonBytes, bytes []byte, err error) {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, nil, fmt.Errorf("readall: %v", err)
	}

	return bytesOrJSON(b)
}

func bytesOrJSON(b []byte) (jsonBytes, rawBytes []byte, err error) {
	var indentedJSON bytes.Buffer
	if err := json.Indent(&indentedJSON, b, "", "  "); err != nil {
		return nil, b, nil
	}
	return indentedJSON.Bytes(), nil, nil
}
