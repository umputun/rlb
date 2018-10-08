package picker

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/umputun/rlb/app/config"
)

func TestRandom_Pick(t *testing.T) {

	ts1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Logf("request %+v", r)
		if r.Method == "GET" && r.URL.Path == "/test/good_get1" {
			fmt.Fprintln(w, "good get 1")
			return
		}
		if r.Method == "GET" && r.URL.Path == "/test/good_get2" {
			fmt.Fprintln(w, "good get 2")
			return
		}
		w.WriteHeader(http.StatusBadRequest)
	}))

	defer ts1.Close()

	ts2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Logf("request %+v", r)
		if r.Method == "GET" && r.URL.Path == "/test/good_get1" {
			fmt.Fprintln(w, "good get 1")
			return
		}
		if r.Method == "GET" && r.URL.Path == "/test/good_get2" {
			fmt.Fprintln(w, "good get 2")
			return
		}
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer ts2.Close()

	nmap := map[string][]config.Node{
		"test": []config.Node{
			config.Node{Server: ts1.URL, Method: "GET", Ping: "/test/good_get1", Weight: 1},
			config.Node{Server: ts2.URL, Method: "GET", Ping: "/test/good_get1", Weight: 1},
		},
	}
	rw := NewRandomWeighted(config.NodesMap(nmap), time.Second, time.Millisecond*100)
	time.Sleep(2 * time.Second)

	r, _, err := rw.Pick("test", "/test/good_get1")
	assert.NoError(t, err)
	assert.True(t, strings.HasPrefix(r, ts1.URL) || strings.HasPrefix(r, ts2.URL))
	assert.True(t, strings.HasSuffix(r, "/test/good_get1"))
}
