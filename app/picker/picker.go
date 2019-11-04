// Package picker gets list of healthy nodes and pick one of them randomly based on weight
package picker

import (
	"net/http"
	"time"

	log "github.com/go-pkgz/lgr"
	"github.com/pkg/errors"

	"github.com/umputun/rlb/app/config"
)

// Node has a part from config and alive + changed for status monitoring
type Node struct {
	config.Node
	alive   bool
	changed bool
}

// nodesFromConf makes picker Node from config
func nodesFromConf(nodes config.NodesMap) (result map[string][]Node) {
	result = map[string][]Node{}
	for k, v := range nodes {
		result[k] = []Node{}
		for _, n := range v {
			result[k] = append(result[k], Node{Node: n})
		}
	}
	return result
}

// checkURL with given method
func checkURL(url, method string, timeout time.Duration) error {

	var resp *http.Response
	var err error

	client := http.Client{Timeout: timeout}
	switch method {
	case "HEAD", "":
		resp, err = client.Head(url)
	case "GET":
		resp, err = client.Get(url)
	default:
		return errors.Errorf("refused to hit %s, unknown method %s", url, method)
	}

	if err != nil {
		return errors.Wrapf(err, "failed to hit %s, method %s", url, method)
	}

	defer func() {
		if e := resp.Body.Close(); e != nil {
			log.Printf("[WARN] failed to close response body, %v", e)
		}
	}()

	if resp.StatusCode >= 400 {
		return errors.Errorf("bad status code %d for %s", resp.StatusCode, url)
	}

	return nil
}

func getCounts(nodes []Node) (good, bad int) {
	for _, n := range nodes {
		if n.alive {
			good++
		} else {
			bad++
		}
	}
	return good, bad
}
