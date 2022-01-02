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

func TestRandom_PickNoFailBack(t *testing.T) {

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
		"test": {
			{Server: ts1.URL, Method: "GET", Ping: "/test/good_get1", Weight: 1},
			{Server: ts2.URL, Method: "GET", Ping: "/test/good_get1", Weight: 1},
		},
	}
	rw := NewRandomWeighted(config.NodesMap(nmap), time.Second, time.Millisecond*100, "")
	time.Sleep(2 * time.Second)

	r, _, err := rw.Pick("test", "/test/good_get1")
	assert.NoError(t, err)
	assert.True(t, strings.HasPrefix(r, ts1.URL) || strings.HasPrefix(r, ts2.URL))
	assert.True(t, strings.HasSuffix(r, "/test/good_get1"))
}

func TestRandom_PickWithFailBack(t *testing.T) {

	calls := 0
	ts1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Logf("request %+v", r)
		if r.Method == "HEAD" {
			calls++
			if calls > 1 {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusOK)
			return
		}
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

		if r.Method == "HEAD" {
			calls++
			if calls > 1 {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusOK)
			return
		}

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
		"test": {
			{Server: ts1.URL, Method: "GET", Ping: "/test/good_get1", Weight: 1},
			{Server: ts2.URL, Method: "GET", Ping: "/test/good_get1", Weight: 1},
		},
	}
	rw := NewRandomWeighted(config.NodesMap(nmap), time.Second, time.Millisecond*100, "http://archive.example.com/media")
	time.Sleep(2 * time.Second)

	{
		r, _, err := rw.Pick("test", "/test/good_get1")
		assert.NoError(t, err)
		assert.True(t, strings.HasPrefix(r, ts1.URL) || strings.HasPrefix(r, ts2.URL))
		assert.True(t, strings.HasSuffix(r, "/test/good_get1"))
	}

	{
		r, _, err := rw.Pick("test", "/test/good_get1")
		assert.NoError(t, err)
		assert.Equal(t, "http://archive.example.com/media/test/good_get1", r)
	}

}
