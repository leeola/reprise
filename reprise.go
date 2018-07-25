package reprise

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
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

type stepFmt struct {
	Step    int
	Method  string
	URLPath string
}

type Reprise struct {
	mu        sync.Mutex
	path      string
	steps     []*stepFmt
	stepIndex int
}

func New(path string) (*Reprise, error) {
	rep := &Reprise{
		path: path,
	}

	// set will ensure we have at least a slice of one, regardless of
	// what was loaded from the directory.
	rep.SetStep(0)

	return rep, nil
}

func (rep *Reprise) Step() (int, *Response, *Request, error) {
	rep.mu.Lock()
	i := rep.stepIndex
	step := rep.steps[i]
	rep.mu.Unlock()

	if step == nil {
		return i, nil, nil, nil
	}

	reqPath := filepath.Join(rep.path, step.RequestFilename())
	b, err := ioutil.ReadFile(reqPath)
	if err != nil && !os.IsNotExist(err) {
		return 0, nil, nil, fmt.Errorf("request readfile: %v", err)
	}

	var req *Request
	if b != nil {
		if err := json.Unmarshal(b, req); err != nil {
			return 0, nil, nil, fmt.Errorf("request unmarshal: %v", err)
		}
	}

	resPath := filepath.Join(rep.path, step.ResponseFilename())
	b, err = ioutil.ReadFile(resPath)
	if err != nil && !os.IsNotExist(err) {
		return 0, nil, nil, fmt.Errorf("response readfile: %v", err)
	}

	var res *Response
	if b != nil {
		if err := json.Unmarshal(b, res); err != nil {
			return 0, nil, nil, fmt.Errorf("response unmarshal: %v", err)
		}
	}

	return i, res, req, nil
}

func (rep *Reprise) MakeRequest() (Response, error) {
	return Response{}, errors.New("not implemented")
}

func (rep *Reprise) DiffReprise() ([]string, error) {
	return nil, errors.New("not implemented")
}

func (rep *Reprise) verifyStep(method, urlStr string) (stepFmt, error) {
	step := rep.steps[rep.stepIndex]

	newStep := step == nil

	url, err := url.Parse(urlStr)
	if err != nil {
		return stepFmt{}, fmt.Errorf("url parse: %v", err)
	}

	urlPath := url.Path

	if !newStep {
		switch {
		case step.Method != method:
			return stepFmt{}, fmt.Errorf("cannot write multiple methods for step: %s", step)
		case step.URLPath != urlPath:
			return stepFmt{}, fmt.Errorf("cannot write multiple urls for step: %s", step)
		default:
			return *step, nil
		}
	}

	step = &stepFmt{
		Step:    rep.stepIndex,
		Method:  method,
		URLPath: urlPath,
	}

	rep.steps[rep.stepIndex] = step

	return *step, nil
}

func (rep *Reprise) Write(res Response, req Request) error {
	rep.mu.Lock()
	defer rep.mu.Unlock()

	step, err := rep.verifyStep(req.Method, req.URI)
	if err != nil {
		return err // no wrap
	}

	b, err := json.MarshalIndent(req, "", "  ")
	if err != nil {
		return fmt.Errorf("request marshalindent: %v", err)
	}

	path := filepath.Join(rep.path, step.RequestFilename())
	if err := ioutil.WriteFile(path, b, 0644); err != nil {
		return fmt.Errorf("request writefile: %v", err)
	}

	b, err = json.MarshalIndent(res, "", "  ")
	if err != nil {
		return fmt.Errorf("response marshalindent: %v", err)
	}

	path = filepath.Join(rep.path, step.ResponseFilename())
	if err := ioutil.WriteFile(path, b, 0644); err != nil {
		return fmt.Errorf("response writefile: %v", err)
	}

	// increment the step index
	rep.next()

	return nil
}

func (rep *Reprise) SetStep(i int) {
	rep.mu.Lock()
	defer rep.mu.Unlock()

	stepsLen := len(rep.steps)
	if i >= stepsLen {
		appendTotal := stepsLen - i + 1
		rep.steps = append(rep.steps, make([]*stepFmt, appendTotal)...)
	}

	rep.stepIndex = i
}

func (rep *Reprise) next() {
	rep.stepIndex++
	stepsLen := len(rep.steps)

	// if step index is smaller than the total steps,
	// try and find the next non-nil step. This may not find
	// any non-nil step, in which case it falls through to the
	// append statement below.
	if rep.stepIndex < stepsLen {
		for i := rep.stepIndex; i < stepsLen; i++ {
			if rep.steps[i] == nil {
				continue
			}

			rep.stepIndex = i
			return
		}
	}

	rep.steps = append(rep.steps, nil)
	rep.stepIndex = stepsLen
}

func (s stepFmt) String() string {
	return fmt.Sprintf("step(%02d, %s, %s)", s.Step, s.Method, s.URLPath)
}

func (s stepFmt) filename() string {
	// NOTE(leeola): using ToLower on the url means that Reprise will
	// consider the same url characters but different case as the same URL.
	// This normalization is to support the fact that OSX and windows do not
	// respect case.
	//
	// Since these reprise files on disk are likely to be committed to git
	// and run by multiple OSs, we have to appease the lowest common
	// denominator. Which means, no case support for URLs, unfortunately.
	url := strings.ToLower(urlCharsReg.ReplaceAllString(s.URLPath, "_"))
	return fmt.Sprintf("%02d.%s.%s", s.Step, strings.ToLower(s.Method), url)
}

func (s stepFmt) RequestFilename() string {
	return fmt.Sprintf("%s.request.json", s.filename())
}

func (s stepFmt) ResponseFilename() string {
	return fmt.Sprintf("%s.response.json", s.filename())
}
