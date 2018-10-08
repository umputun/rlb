package picker

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCheckURL(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Logf("request %+v", r)
		if r.Method == "GET" && r.URL.Path == "/good_get" {
			fmt.Fprintln(w, "good get")
			return
		}
		if r.Method == "GET" && r.URL.Path == "/slow_get" {
			time.Sleep(1 * time.Second)
			fmt.Fprintln(w, "slow get")
			return
		}
		if r.Method == "HEAD" && r.URL.Path == "/good_head" {
			fmt.Fprintln(w, "good head")
			return
		}
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer ts.Close()

	tbl := []struct {
		url     string
		method  string
		isError bool
	}{
		{"/good_get", "GET", false},
		{"/good_head", "HEAD", false},
		{"/blah", "HEAD", true},
		{"/blah", "GET", true},
		{"/slow_get", "GET", true},
		{"/good_get", "POST", true},
	}

	for i, tt := range tbl {
		err := checkURL(ts.URL+tt.url, tt.method, time.Millisecond*500)
		if tt.isError {
			assert.NotNil(t, err, "check #%d", i)
			continue
		}
		assert.NoError(t, err, "check #%d", i)
	}
}
