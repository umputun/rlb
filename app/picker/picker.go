package picker

import (
	"log"
	"net/http"
	"time"

	"github.com/pkg/errors"

	"github.com/umputun/rlb/app/config"
)

// Interface defines pick method to return final redirect url from servcie and resource
type Interface interface {
	Pick(svc string, resource string) (resURL string, node Node, err error)
	Nodes() map[string][]Node
}

// Node has a part from config and alive + changed for status monitoring
type Node struct {
	config.Node
	alive   bool
	changed bool
}

// nodesFromConf makes picker Node from config
func nodesFromConf(nodes map[string][]config.Node) (result map[string][]Node) {
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
func checkURL(URL string, method string, timeout time.Duration) error {

	var resp *http.Response
	var err error

	client := http.Client{Timeout: timeout}
	switch method {
	case "HEAD", "":
		resp, err = client.Head(URL)
	case "GET":
		resp, err = client.Get(URL)
	default:
		return errors.Wrapf(err, "refused to hit %s, unknown method %s", URL, method)
	}

	if err != nil {
		return errors.Wrapf(err, "failed to hit %s, method %s", URL, method)
	}

	defer func() {
		if e := resp.Body.Close(); e != nil {
			log.Printf("[WARN] failed to close response body, %v", e)
		}
	}()

	if resp.StatusCode >= 400 {
		return errors.Errorf("bad status code %d for %s", resp.StatusCode, URL)
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
