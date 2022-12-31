// Package server implements rest server handling  "jump" requests and optionally updating stats service
package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/didip/tollbooth/v7"
	"github.com/didip/tollbooth_chi"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	log "github.com/go-pkgz/lgr"
	"github.com/go-pkgz/rest"
	"github.com/go-pkgz/rest/logger"
	"github.com/lithammer/shortuuid/v4"
	"github.com/umputun/rlb/app/picker"
)

// RLBServer - main rlb server
type RLBServer struct {
	nodePicker Picker
	statsURL   string
	errMsg     string
	version    string
	port       int
	bench      *rest.Benchmarks
	httpServer *http.Server
	lock       sync.Mutex
}

// Picker defines pick method to return final redirect url from service and resource
type Picker interface {
	Pick(svc string, resource string) (resURL string, node picker.Node, err error)
	Nodes() map[string][]picker.Node
	Status() (bool, []string)
}

// LogRecord for stats
type LogRecord struct {
	ID       string    `json:"id,omitempty"`
	FromIP   string    `json:"from_ip"`
	TS       time.Time `json:"ts,omitempty"`
	FileName string    `json:"file_name"`
	Service  string    `json:"service"`
	DestHost string    `json:"dest"`
	Referer  string    `json:"referer"`
}

// NewRLBServer makes a new rlb server for map of services
func NewRLBServer(nodePicker Picker, emsg, statsURL string, port int, version string) *RLBServer {
	res := RLBServer{
		nodePicker: nodePicker,
		errMsg:     emsg,
		statsURL:   statsURL,
		version:    version,
		port:       port,
		bench:      rest.NewBenchmarks(),
	}
	for k, v := range nodePicker.Nodes() {
		log.Printf("[INFO] service=%s, nodes=%v", k, v)
	}
	return &res
}

// Run activates alive updater and web server
func (s *RLBServer) Run() {
	log.Printf("[INFO] activate web server on port %d", s.port)
	router := s.routes()

	s.lock.Lock()
	s.httpServer = &http.Server{
		Addr:              fmt.Sprintf(":%d", s.port),
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      5 * time.Second,
		IdleTimeout:       30 * time.Second,
	}
	s.lock.Unlock()

	err := s.httpServer.ListenAndServe()
	log.Printf("[WARN] http server terminated, %s", err)
}

// Shutdown rlb http server
func (s *RLBServer) Shutdown() {
	log.Print("[WARN] shutdown rest server")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	s.lock.Lock()
	if s.httpServer != nil {
		if err := s.httpServer.Shutdown(ctx); err != nil {
			log.Printf("[DEBUG] http shutdown error, %s", err)
		}
		log.Print("[DEBUG] shutdown http server completed")
	}
	s.lock.Unlock()
}

func (s *RLBServer) routes() chi.Router {

	router := chi.NewRouter()

	router.Use(middleware.RequestID, middleware.RealIP, rest.Recoverer(log.Default()))
	router.Use(middleware.Throttle(10000), middleware.Timeout(60*time.Second))
	router.Use(rest.AppInfo("RLB", "Umputun", s.version), rest.Ping)
	router.Use(tollbooth_chi.LimitHandler(tollbooth.NewLimiter(50, nil)), middleware.NoCache)

	router.Use(logger.New(logger.Log(log.Default()), logger.WithBody, logger.Prefix("[DEBUG]"),
		logger.IPfn(logger.AnonymizeIP)).Handler)
	router.Use()

	// current routes
	router.Route("/api/v1/jump", func(r chi.Router) {
		r.Use(s.bench.Handler)
		r.Get("/{svc}", s.DoJump)
		r.Head("/{svc}", s.DoJump)
	})

	// legacy routes
	router.Group(func(r chi.Router) {
		r.Use(s.bench.Handler)
		r.Get("/{svc}", s.DoJump)
		r.Head("/{svc}", s.DoJump)
	})

	router.Get("/api/v1/status", s.statusCtrl)
	router.Get("/api/v1/bench", s.benchCtrl)

	return router
}

// DoJump - jump to alive server for svc, url = Query("url")
func (s *RLBServer) DoJump(w http.ResponseWriter, r *http.Request) {
	svc := chi.URLParam(r, "svc")
	url := r.URL.Query().Get("url")
	log.Printf("[DEBUG] jump %s %s", svc, url)
	redirURL, node, err := s.nodePicker.Pick(svc, url)
	if err != nil {
		render.Status(r, http.StatusNotFound)
		render.HTML(w, r, s.errMsg)
		return
	}

	log.Printf("[DEBUG] redirect to %s%s", node.Server, url)
	go func() {
		if err := s.submitStats(r, node, svc+url); err != nil {
			log.Printf("[DEBUG] can't submit stats, %s", err)
		}
	}()

	http.Redirect(w, r, redirURL, http.StatusFound)
}

func (s *RLBServer) submitStats(r *http.Request, node picker.Node, url string) error {
	if s.statsURL == "" {
		return nil
	}

	fileNameSplit := strings.Split(strings.TrimLeft(url, "/"), "/")
	lrec := LogRecord{
		ID:       shortuuid.New(),
		FromIP:   strings.Split(r.RemoteAddr, ":")[0],
		TS:       time.Now(),
		FileName: strings.Join(fileNameSplit[1:], "/"),
		Service:  fileNameSplit[0],
		DestHost: strings.TrimPrefix(strings.TrimPrefix(node.Server, "http://"), "https://"),
		Referer:  r.Referer(),
	}
	client := http.Client{Timeout: time.Millisecond * 100}

	data, err := json.Marshal(&lrec)
	if err != nil {
		return fmt.Errorf("can't marshal: %w", err)
	}

	resp, err := client.Post(s.statsURL, "application/json", bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("remote call failed: %w", err)
	}

	defer func() {
		if e := resp.Body.Close(); e != nil {
			log.Printf("[WARN] failed to close response body, %v", e)
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[WARN] failed to read response body, %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status code %d, body %s", resp.StatusCode, string(body))
	}

	return nil
}

// GET /api/v1/status - returns status of all nodes, 200, 417 failed
func (s *RLBServer) statusCtrl(w http.ResponseWriter, r *http.Request) {
	ok, failed := s.nodePicker.Status()
	if !ok {
		render.Status(r, http.StatusExpectationFailed)
		render.JSON(w, r, rest.JSON{"status": "failed", "hosts": failed})
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, rest.JSON{"status": "ok"})
}

// GET /api/v1/bench - returns benchmarks json for 1, 5 and 15 minutes ranges
func (s *RLBServer) benchCtrl(w http.ResponseWriter, r *http.Request) {
	resp := struct {
		OneMin     rest.BenchmarkStats `json:"1min"`
		FiveMin    rest.BenchmarkStats `json:"5min"`
		FifteenMin rest.BenchmarkStats `json:"15min"`
	}{
		s.bench.Stats(time.Minute),
		s.bench.Stats(time.Minute * 5),
		s.bench.Stats(time.Minute * 15),
	}

	render.JSON(w, r, resp)
}
