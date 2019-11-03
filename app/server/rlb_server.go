package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/didip/tollbooth"
	"github.com/didip/tollbooth_chi"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
	log "github.com/go-pkgz/lgr"
	"github.com/go-pkgz/rest"
	"github.com/go-pkgz/rest/logger"
	"github.com/lithammer/shortuuid"
	"github.com/pkg/errors"

	"github.com/umputun/rlb/app/picker"
)

// RLBServer - main rlb server
type RLBServer struct {
	picker   picker.Interface
	statsURL string
	errMsg   string
	version  string
	port     int

	httpServer *http.Server
	lock       sync.Mutex
}

// LogRecord for stats
type LogRecord struct {
	ID       string    `json:"id,omitempty"`
	FromIP   string    `json:"from_ip"`
	TS       time.Time `json:"ts,omitempty"`
	FileName string    `json:"file_name"`
	Service  string    `json:"service"`
	DestHost string    `json:"dest"`
}

// NewRLBServer makes a new rlb server for map of services
func NewRLBServer(picker picker.Interface, emsg, statsURL string, port int, version string) *RLBServer {
	res := RLBServer{
		picker:   picker,
		errMsg:   emsg,
		statsURL: statsURL,
		version:  version,
		port:     port,
	}
	for k, v := range picker.Nodes() {
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
	router.Use(middleware.Throttle(1000), middleware.Timeout(60*time.Second))
	router.Use(rest.AppInfo("RLB", "Umputun", s.version), rest.Ping)
	router.Use(tollbooth_chi.LimitHandler(tollbooth.NewLimiter(50, nil)))
	router.Use(logger.New(logger.Log(log.Default()), logger.WithBody, logger.Prefix("[INFO]")).Handler)

	// current routes
	router.Route("/api/v1/jump", func(r chi.Router) {
		r.Get("/{svc}", s.DoJump)
		r.Head("/{svc}", s.DoJump)
	})

	// legacy routes
	router.Get("/{svc}", s.DoJump)
	router.Head("/{svc}", s.DoJump)

	return router
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
			log.Printf("[DEBUG] can't submit stats, %s", err)
		}
	}()

	http.Redirect(w, r, redirurl, 302)
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
