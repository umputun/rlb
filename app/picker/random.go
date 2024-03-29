package picker

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	log "github.com/go-pkgz/lgr"

	"github.com/umputun/rlb/app/config"
)

// RandomWeighted implements picker with the random, weighted selection
type RandomWeighted struct {
	refresh     time.Duration
	timeout     time.Duration
	failBackURL string
	nodes       map[string][]Node
	lock        sync.RWMutex
}

// NewRandomWeighted makes new picker. Activate alive update thread
func NewRandomWeighted(nodes config.NodesMap, refresh, timeout time.Duration, failBackURL string) *RandomWeighted {
	res := RandomWeighted{nodes: nodesFromConf(nodes), refresh: refresh, timeout: timeout, failBackURL: failBackURL}
	go res.updateAlive()
	log.Printf("[DEBUG] nodes %+v", nodes)
	return &res
}

// Pick random node with weights
func (w *RandomWeighted) Pick(svc, resource string) (resURL string, node Node, err error) {
	log.Printf("[DEBUG] pick %s for %s", svc, resource)

	w.lock.RLock()
	defer w.lock.RUnlock()

	alive := []Node{}

	// get alive-only nodes for svc, multiple by Weight count
	for _, node := range w.nodes[svc] {
		if node.alive && node.Weight > 0 {
			for i := 0; i < node.Weight; i++ {
				alive = append(alive, node)
			}
		}
	}

	if len(alive) == 0 {
		return "", Node{}, fmt.Errorf("no node for %s", svc)
	}

	node = alive[rand.Intn(len(alive))] // nolint

	resURL = node.Server + resource
	if w.failBackURL != "" {
		if err = checkURL(resURL, "HEAD", w.timeout); err != nil {
			resURL = w.failBackURL + resource
		}
	}

	return resURL, node, nil
}

// Nodes return list of all current nodes
func (w *RandomWeighted) Nodes() map[string][]Node {
	w.lock.RLock()
	defer w.lock.RUnlock()
	return w.nodes
}

// Status return status of all nodes, true if all nodes are alive, false if at least one is dead and return list of dead nodes
func (w *RandomWeighted) Status() (ok bool, failed []string) {
	w.lock.RLock()
	defer w.lock.RUnlock()

	for _, nodes := range w.nodes {
		for _, node := range nodes {
			if !node.alive {
				failed = append(failed, node.Server)
			}
		}
	}
	return len(failed) == 0, failed
}

// updateAlive runs periodic pings to all nodes, updates nodes
func (w *RandomWeighted) updateAlive() {
	log.Printf("[DEBUG] alive updater started. refresh=%v, socket timeout=%v", w.refresh, w.timeout)

	// update alive status for svc, tests all nodes in parallel
	update := func(svc string) int {
		respCh := make(chan Node, len(w.nodes[svc]))
		updNodes := make([]Node, 0, len(w.nodes[svc]))
		changed := 0

		for _, n := range w.nodes[svc] {
			go func(node Node) {
				checkedNode := node
				pingURL := fmt.Sprintf("%s%s", node.Server, node.Ping)
				err := checkURL(pingURL, node.Method, w.timeout)
				if err != nil {
					log.Printf("[DEBUG] %v", err)
				}
				checkedNode.alive = err == nil
				checkedNode.changed = checkedNode.alive != node.alive
				if checkedNode.changed {
					log.Printf("[INFO] changed status of %s [%s], %v -> %v", node.Server, svc, node.alive, checkedNode.alive)
					if err != nil {
						log.Printf("[INFO] %v", err)
					}
				}
				respCh <- checkedNode
			}(n)
		}

		for i := 0; i < len(w.nodes[svc]); i++ {
			checkedNode := <-respCh
			updNodes = append(updNodes, checkedNode)
			if checkedNode.changed {
				changed++
			}
		}

		w.lock.Lock()
		copy(w.nodes[svc], updNodes)
		w.lock.Unlock()

		return changed
	}

	for {
		for k := range w.nodes {
			if changed := update(k); changed > 0 {
				good, bad := getCounts(w.nodes[k])
				log.Printf("[INFO] %s alive counts updated, changed=%d {total:%d, passed:%d, failed:%d}",
					k, changed, good+bad, good, bad)
			}
		}
		time.Sleep(w.refresh)
	}
}
