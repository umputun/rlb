package config

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGet(t *testing.T) {
	conf := NewConf(strings.NewReader(rlbYaml))
	r := conf.Get()
	assert.Equal(t, 2, len(r), "2 services in the map")
	assert.Equal(t, 3, len(r["test1"]), "3 nodes in test1")

	assert.Equal(t, "HEAD", r["test1"][0].Method, "HEAD as a default method")
	assert.Equal(t, "HEAD", r["test1"][1].Method, "HEAD as explicit method")
	assert.Equal(t, "GET", r["test1"][2].Method, "GET as explicit method")
	assert.Equal(t, "blah", conf.NoNode.Message)
	assert.Equal(t, "http://archive.radio-t.com/media", conf.FailBackURL)
}

const rlbYaml = `
services:
 test1:
  - server: http://n1.radio-t.com
    ping: /rtfiles/rt_podcast480.mp3
    weight: 1

  - server: http://n2.radio-t.com
    ping: /rtfiles/rt_podcast480.mp3
    method: HEAD
    weight: 1

  - server: http://n3.radio-t.com
    ping: /rtfiles/rt_podcast480.mp3
    method: GET
    weight: 5

 test2:
  - server: http://n5.radio-t.com
    ping: /rtfiles/rt_podcast480.mp3
    method: GET
    weight: 1

  - server: http://n2.radio-t.com
    ping: /rtfiles/rt_podcast480.mp3
    method: GET
    weight: 3

no_node:
 message: blah

failback: http://archive.radio-t.com/media
`
