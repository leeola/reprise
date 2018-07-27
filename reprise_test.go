package reprise_test

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/leeola/reprise"
)

const tmpRoot = "testdata/tmp"

func TestRepriseDiff(t *testing.T) {
	os.MkdirAll(tmpRoot, 0755)
	p, err := ioutil.TempDir(tmpRoot, "write")
	if err != nil {
		t.Fatalf("tempdir: %v", err)
	}
	defer os.RemoveAll(p)

	rep, err := reprise.New(p)
	if err != nil {
		t.Fatalf("reprise new: %v", err)
	}

	m := reprise.Middleware(rep)

	responseStr := `foo`
	h := m(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, responseStr)
	}))

	ts := httptest.NewServer(h)
	defer ts.Close()

	// call get to make the middleware save the request/response
	http.Get(ts.URL)
	rep.SetStep(0)

	// change the http response
	responseStr = "baz"

	// play the last request again
	diffs, err := rep.RepriseDiff(ts.URL)
	if err != nil {
		t.Fatalf("reprisediff: %v", err)
	}
	rep.SetStep(0)

	if len(diffs) == 0 {
		t.Errorf("no binary diffs returned")
	}

	if len(diffs) > 1 {
		t.Errorf("unexpected binary diff length")
	}

	if want := "binary bytes do not match"; diffs[0] != want {
		t.Errorf("unexpected diff msg. want:%q, got:%q", want, diffs[0])
	}

	// change to json http response
	responseStr = `{"foo": "bar"}`

	// call get to make the middleware save the request/response
	http.Get(ts.URL)
	rep.SetStep(0)

	// change response for diff
	responseStr = `{"foo": "baz"}`

	diffs, err = rep.RepriseDiff(ts.URL)
	if err != nil {
		t.Fatalf("reprisediff: %v", err)
	}
	rep.SetStep(0)

	if len(diffs) == 0 {
		t.Errorf("no json diffs returned")
	}

	if len(diffs) > 1 {
		t.Errorf("unexpected json diff length")
	}

	if want := "map[foo]: baz != bar"; diffs[0] != want {
		t.Errorf("unexpected diff msg. want:%q, got:%q", want, diffs[0])
	}
}
