package reprise

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/leeola/reprise/jsondiff"
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

	// setting the step to 0 ensures we initialize the steps slice.
	rep.setStep(0)

	return rep, nil
}

func NewMkdir(path string, perm os.FileMode) (*Reprise, error) {
	if err := os.Mkdir(path, perm); err != nil {
		return nil, err // no wrap
	}

	return New(path)
}

func (rep *Reprise) Step() (int, *Response, *Request, error) {
	rep.mu.Lock()
	defer rep.mu.Unlock()

	i := rep.stepIndex
	step := rep.steps[i]

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
		var r Request
		if err := json.Unmarshal(b, &r); err != nil {
			return 0, nil, nil, fmt.Errorf("request unmarshal: %v", err)
		}
		req = &r
	}

	resPath := filepath.Join(rep.path, step.ResponseFilename())
	b, err = ioutil.ReadFile(resPath)
	if err != nil && !os.IsNotExist(err) {
		return 0, nil, nil, fmt.Errorf("response readfile: %v", err)
	}

	var res *Response
	if b != nil {
		var r Response
		if err := json.Unmarshal(b, &r); err != nil {
			return 0, nil, nil, fmt.Errorf("response unmarshal: %v", err)
		}
		res = &r
	}

	return i, res, req, nil
}

func (rep *Reprise) reprise(url string, req Request) (Response, error) {
	httpReq, err := req.HTTPRequest(url)
	if err != nil {
		return Response{}, fmt.Errorf("httprequest: %v", err)
	}

	c := &http.Client{}
	resp, err := c.Do(httpReq)
	if err != nil {
		return Response{}, fmt.Errorf("http do: %v", err)
	}

	return NewResponse(resp)
}

// RepriseDiff creates a http request to the given url combining it with
// the stored Request.URI for this step and returns diffs of the response.
func (rep *Reprise) RepriseDiff(url string) ([]string, error) {
	_, stepRes, stepReq, err := rep.Step()
	if err != nil {
		return nil, fmt.Errorf("read step: %v", err)
	}

	if stepReq == nil {
		return nil, ErrNoRequest
	}
	if stepReq == nil {
		return nil, ErrNoResponse
	}

	got, err := rep.reprise(url, *stepReq)
	if err != nil {
		return nil, err // no wrap
	}

	want := *stepRes

	if got.IsJSON() != want.IsJSON() {
		// TODO(leeola): make a diff msg similar to whatever i'm using
		// for the diffing algo.
		return []string{"two different response bodies, json and non-json"}, nil
	}

	if !got.IsJSON() {
		if len(got.BodyBinary) != len(want.BodyBinary) {
			// TODO(leeola): make a diff msg similar to whatever i'm using
			// for the diffing algo.
			return []string{"binary bytes length does not match"}, nil
		}

		for i, wantB := range want.BodyBinary {
			gotB := got.BodyBinary[i]
			if wantB != gotB {
				// TODO(leeola): make a diff msg similar to whatever i'm using
				// for the diffing algo.
				return []string{"binary bytes do not match"}, nil
			}
		}
	} else {
		diff, err := jsondiff.Diff(got.BodyJSON, want.BodyJSON)
		if err != nil {
			return nil, fmt.Errorf("jsondiff: %v", err)
		}

		return diff, nil
	}

	return nil, nil
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

	rep.setStep(i)
}

// setStep is like SetStep, but without a lock, usable from locking
// methods.
func (rep *Reprise) setStep(i int) {
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

	if rep.stepIndex >= stepsLen {
		rep.steps = append(rep.steps, nil)
		rep.stepIndex = stepsLen
	}
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
