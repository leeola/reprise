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

func TestMiddleware(t *testing.T) {
	os.MkdirAll(tmpRoot, 0755)
	p, err := ioutil.TempDir(tmpRoot, "write")
	if err != nil {
		t.Fatalf("tempdir: %v", err)
	}
	// defer os.RemoveAll(p)

	rep, err := reprise.New(p)
	if err != nil {
		t.Fatalf("reprise new: %v", err)
	}

	m := reprise.Middleware(rep)

	h := m(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Hello, client")
	}))

	ts := httptest.NewServer(h)
	defer ts.Close()

	resp, err := http.Get(ts.URL)
	if err != nil {
		t.Fatalf("get: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("unexpected status: %d, %q", resp.StatusCode, resp.Status)
	}

	_, err = rep.Request()
	if err != nil {
		t.Errorf("reprise request: %v", err)
	}

	// resp, err = http.Post(ts.URL, "", strings.NewReader(`{"foo": "bar"}`))
	// if err != nil {
	// 	t.Fatalf("post: %v", err)
	// }

	// if resp.StatusCode != http.StatusOK {
	// 	t.Errorf("unexpected status: %d, %q", resp.StatusCode, resp.Status)
	// }
}
