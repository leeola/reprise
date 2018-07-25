package reprise

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	// urlCharsReg is used to remove non-filesystem friendly characters from
	// a url path.
	//
	// Ie, / signals a directory on unix, so naming a file with a / char
	// causes annoyances.
	//
	// Example:
	//    /foo/bar  -> _foo_bar
	//    /         -> _
	urlCharsReg *regexp.Regexp
)

func init() {
	r, err := regexp.Compile("[^a-zA-Z0-9]+")
	if err != nil {
		panic(fmt.Sprintf("urlCharsReg compile: %v", err))
	}
	urlCharsReg = r
}

type Step struct {
	Step   int
	Method string
	URL    string
}

type Request struct {
	Method string          `json:"method"`
	JSON   json.RawMessage `json:"json,omitempty"`
	Body   []byte          `json:"body,omitempty"`
}

type Response struct {
	Body json.RawMessage `json:"body"`
}

type Reprise struct {
	path      string
	steps     []*Step
	stepIndex int
}

func New(path string) (*Reprise, error) {
	return &Reprise{
		path: path,
	}, nil
}

func (rep *Reprise) Request() (Request, error) {
	step := rep.step()

	stepPath := filepath.Join(rep.path, step.RequestFilename())
	b, err := ioutil.ReadFile(stepPath)
	if err != nil {
		return Request{}, fmt.Errorf("readfile: %v", err)
	}

	var r Request
	if err := json.Unmarshal(b, &r); err != nil {
		return Request{}, fmt.Errorf("unmarshal: %v", err)
	}

	return r, nil
}

func (rep *Reprise) Response() (Response, error) {
	return Response{}, errors.New("not implemented")
}

func (rep *Reprise) MakeRequest() (Response, error) {
	return Response{}, errors.New("not implemented")
}

func (rep *Reprise) DiffReprise() ([]string, error) {
	return nil, errors.New("not implemented")
}

func (rep *Reprise) step() *Step {
	stepsLen := len(rep.steps)
	if rep.stepIndex >= stepsLen {
		appendTotal := rep.stepIndex - stepsLen + 1
		rep.steps = append(rep.steps, make([]*Step, appendTotal)...)
	}

	return rep.steps[rep.stepIndex]
}

func (rep *Reprise) makeStep(method, url string) (Step, error) {
	name := rep.step()

	newStep := name == nil

	if !newStep {
		switch {
		case name.Method != method:
			return Step{}, fmt.Errorf("cannot write over existing step: %s", name)
		case name.URL != url:
			return Step{}, fmt.Errorf("cannot write over existing step: %s", name)
		default:
			return *name, nil
		}
	}

	name = &Step{
		Step:   rep.stepIndex,
		Method: method,
		URL:    url,
	}

	rep.steps[rep.stepIndex] = name

	return *name, nil
}

func (rep *Reprise) WriteRequest(httpReq *http.Request) (Request, error) {
	r := Request{
		Method: httpReq.Method,
	}

	if httpReq.Body != nil {
		b, err := ioutil.ReadAll(httpReq.Body)
		if err != nil {
			return Request{}, nil
		}

		var buf bytes.Buffer
		if err := json.Indent(&buf, b, "", "  "); err != nil {
			r.Body = b
		} else {
			r.JSON = buf.Bytes()
		}
	}

	name, err := rep.makeStep(r.Method, httpReq.URL.Path)
	if err != nil {
		return Request{}, err // no wrap
	}

	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return Request{}, fmt.Errorf("marshalindent: %v", err)
	}

	path := filepath.Join(rep.path, name.RequestFilename())

	if err := ioutil.WriteFile(path, b, 0644); err != nil {
		return Request{}, fmt.Errorf("writefile: %v", err)
	}

	return r, nil
}

func (rep *Reprise) WriteResponse() (Response, error) {
	return Response{}, errors.New("not implemented")
}

func (s Step) String() string {
	return s.filename()
}

func (s Step) filename() string {
	// NOTE(leeola): using ToLower on the url means that Reprise will
	// consider the same url characters but different case as the same URL.
	// This normalization is to support the fact that OSX and windows do not
	// respect case.
	//
	// Since these reprise files on disk are likely to be committed to git
	// and run by multiple OSs, we have to appease the lowest common
	// denominator. Which means, no case support for URLs, unfortunately.
	url := strings.ToLower(urlCharsReg.ReplaceAllString(s.URL, "_"))
	return fmt.Sprintf("%02d.%s.%s", s.Step, strings.ToLower(s.Method), url)
}

func (s Step) RequestFilename() string {
	return fmt.Sprintf("%s.request.json", s.filename())
}

func (s Step) ResponseFilename() string {
	return fmt.Sprintf("%s.response.json", s.filename())
}
