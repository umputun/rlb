package server

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/didip/tollbooth"
	"github.com/didip/tollbooth_chi"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
	"github.com/go-pkgz/rest"
	"github.com/go-pkgz/rest/logger"
	"github.com/pkg/errors"

	"github.com/umputun/rlb/app/picker"
)

// RLBServer - main rlb server
type RLBServer struct {
	picker   picker.Interface
	statsURL string
	errMsg   string
	version  string
}

// NewRLBServer makes a new rlb server for map of services
func NewRLBServer(picker picker.Interface, emsg, statsURL, version string) *RLBServer {
	res := RLBServer{
		picker:   picker,
		errMsg:   emsg,
		statsURL: statsURL,
		version:  version,
	}
	for k, v := range picker.Nodes() {
		log.Printf("[INFO] servcie=%s, nodes=%v", k, v)
	}
	return &res
}

// Run activates alive updater and web server
func (s *RLBServer) Run() {
	log.Printf("[INFO] activate web server")
	router := chi.NewRouter()

	router.Use(middleware.RequestID, middleware.RealIP, rest.Recoverer)
	router.Use(middleware.Throttle(1000), middleware.Timeout(60*time.Second))
	router.Use(rest.AppInfo("RLB", "Umputun", s.version), rest.Ping)
	router.Use(tollbooth_chi.LimitHandler(tollbooth.NewLimiter(50, nil)))
	l := logger.New(logger.Flags(logger.All))
	router.Use(l.Handler)

	// legacy routes
	router.Get("/{svc}", s.DoJump)
	router.Head("/{svc}", s.DoJump)

	router.Route("/api/v1/jump", func(r chi.Router) {
		r.Get("/{svc}", s.DoJump)
		r.Head("/{svc}", s.DoJump)
	})

	log.Fatal(http.ListenAndServe(":7070", router))
}

// DoJump - jump to alive server for svc, url = Query("url")
func (s *RLBServer) DoJump(w http.ResponseWriter, r *http.Request) {
	svc := chi.URLParam(r, "svc")
	url := r.URL.Query().Get("url")
	log.Printf("[DEBUG] jump %s %s", svc, url)
	redirurl, node, err := s.picker.Pick(svc, url)
	if err != nil {
		render.Status(r, http.StatusNotFound)
		render.HTML(w, r, s.errMsg)
		return
	}

	log.Printf("[DEBUG] redirect to %s%s", node.Server, url)
	go func() {
		if err := s.submitStats(r, node, svc+url); err != nil {
			log.Printf("[WARN] can't submit stats, %s", err)
		}
	}()

	http.Redirect(w, r, redirurl, 302)
}

func (s *RLBServer) submitStats(r *http.Request, node picker.Node, url string) error {
	if s.statsURL == "" {
		return nil
	}

	// LogRecord for stats
	type LogRecord struct {
		ID       string    `json:"id,omitempty"`
		FromIP   string    `json:"from_ip"`
		TS       time.Time `json:"ts,omitempty"`
		Fname    string    `json:"fname"`
		DestHost string    `json:"dest"`
	}

	lrec := LogRecord{
		FromIP:   strings.Split(r.RemoteAddr, ":")[0],
		Fname:    strings.TrimLeft(url, "/"),
		DestHost: strings.TrimPrefix(strings.TrimPrefix(node.Server, "http://"), "https://"),
	}
	client := http.Client{Timeout: time.Millisecond * 100}

	data, err := json.Marshal(&lrec)
	if err != nil {
		return errors.Wrap(err, "can't marshal")
	}

	resp, err := client.Post(s.statsURL, "application/json", bytes.NewReader(data))
	if err != nil {
		return errors.Wrapf(err, "remote call failed")
	}

	defer func() {
		if e := resp.Body.Close(); e != nil {
			log.Printf("[WARN] failed to close response body, %v", e)
		}
	}()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[WARN] failed to read response body, %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return errors.Errorf("bad status code %d, body %s", resp.StatusCode, string(body))
	}

	return nil
}