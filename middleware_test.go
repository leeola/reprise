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

func TestMiddleware(t *testing.T) {
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

	m := reprise.Middleware(reprise.All(rep))

	responseStr := `{"msg": "Hello world"}`
	h := m(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, responseStr)
	}))

	ts := httptest.NewServer(h)
	defer ts.Close()

	resp, err := http.Get(fmt.Sprintf("%s/foo?bar=baz", ts.URL))
	if err != nil {
		t.Fatalf("get: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("unexpected status: %d, %q", resp.StatusCode, resp.Status)
	}

	if got := string(mustDumpResp(resp)); got != responseStr {
		t.Errorf("unexpected response. want:%q, got:%q", responseStr, got)
	}

	step, _, _, err := rep.Step()
	if err != nil {
		t.Errorf("reprise step: %v", err)
	}

	if step != 1 {
		t.Errorf("unexpected step number. want:%d, got:%d", 1, step)
	}

	resp, err = http.Post(ts.URL, "", strings.NewReader(`{"foo": "bar"}`))
	if err != nil {
		t.Fatalf("post: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("unexpected status: %d, %q", resp.StatusCode, resp.Status)
	}

	if got := string(mustDumpResp(resp)); got != responseStr {
		t.Errorf("unexpected response. want:%q, got:%q", responseStr, got)
	}

	step, _, _, err = rep.Step()
	if err != nil {
		t.Errorf("reprise step: %v", err)
	}

	if step != 2 {
		t.Errorf("unexpected step number. want:%d, got:%d", 2, step)
	}
}

func mustDumpResp(resp *http.Response) []byte {
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(fmt.Sprintf("readall: %v", err))
	}

	return b
}
