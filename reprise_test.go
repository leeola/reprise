package reprise_test

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/leeola/reprise"
)

const tmpRoot = "testdata/tmp"

func TestReprise(t *testing.T) {
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

	resp, err := http.Post(ts.URL, "", strings.NewReader(`{"foo": "bar"}`))
	if err != nil {
		t.Fatalf("post: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("unexpected status: %d, %q", resp.StatusCode, resp.Status)
	}

	if got := string(mustDumpResp(resp)); got != responseStr {
		t.Errorf("unexpected response. want:%q, got:%q", responseStr, got)
	}

	// change the http response
	responseStr = "baz"

	// set the step to the 0th step, as it will have incremented
	// due to writing from the middleware.
	rep.SetStep(0)

	// play the last request again
	diffs, err := rep.RepriseDiff()
	if err != nil {
		t.Fatalf("reprisediff: %v", err)
	}

	fmt.Println("diffs", diffs)
}
