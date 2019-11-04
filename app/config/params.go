// Package config handles params from yml config file
// An example presented in the repo as rlb-sample.yml
package config

import (
	"fmt"
	"io"
	"io/ioutil"

	log "github.com/go-pkgz/lgr"
	yaml "gopkg.in/yaml.v2"
)

// NodesMap wraps map with svc name as a key and all svc nodes as value
type NodesMap map[string][]Node

// ConfFile map by svc for node:conf
type ConfFile struct {
	Services NodesMap `yaml:"services"`
	NoNode   struct {
		Message string `yaml:"message"`
	} `yaml:"no_node"`
}

// Node has a part from config and alive + changed for status monitoring
type Node struct {
	Server string `yaml:"server"`
	Ping   string `yaml:"ping"`
	Weight int    `yaml:"weight"`
	Method string `yaml:"method"`
}

// NewConf makes new config for yml reader
func NewConf(reader io.Reader) *ConfFile {
	res := &ConfFile{}
	data, err := ioutil.ReadAll(reader)
	if err != nil {
		log.Fatalf("[ERROR] failed to read config, %v", err)
	}
	if err = yaml.Unmarshal(data, &res); err != nil {
		log.Fatalf("[ERROR] failed to parse config, %v", err)
	}
	return res
}

// Get map svc:[nodes] and set default method to HEAD (if not defined)
func (c ConfFile) Get() NodesMap {
	res := make(map[string][]Node)
	for service, nodeConf := range c.Services {
		res[service] = []Node{}
		for _, n := range nodeConf {
			if n.Method == "" {
				n.Method = "HEAD"
			}
			res[service] = append(res[service], n)
		}

	}
	return res
}

func (n Node) String() string {
	return fmt.Sprintf("{server:%s, ping:%s, weight:%d, method:%s}", n.Server, n.Ping, n.Weight, n.Method)
}
