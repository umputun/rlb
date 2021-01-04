# RLB - Redirecting Load Balancer 
[![Build Status](https://github.com/umputun/rlb/workflows/build/badge.svg)](https://github.com/umputun/rlb/actions) [![Coverage Status](https://coveralls.io/repos/github/umputun/rlb/badge.svg)](https://coveralls.io/github/umputun/rlb) [![Go Report Card](https://goreportcard.com/badge/github.com/umputun/rlb)](https://goreportcard.com/report/github.com/umputun/rlb) [![Docker Automated build](https://img.shields.io/docker/automated/jrottenberg/ffmpeg.svg)](https://hub.docker.com/r/umputun/rlb/)

This service redirects incoming `GET` and `HEAD` requests (with 302) to the upstream servers. 
Servers picked up randomly, unhealthy boxes excluded dynamically.

_Note: This is not a replacement for reverse proxy. All it does is HTTP redirect, not the real network proxying._

## Install

1. Copy provided `docker-compose.yml`
1. Make `rlb.yml` config for your service(s) (see `rlb-sample.yml` below in [Config file format](#config-file-format) section).
1. Start container with `docker-compose up -d`

This will start rlb on port `:7070` by default and requests like `http://host/api/v1/jump/service1?url=/files/blah.mp3` will be redirected to the one of upstreams.

## API

* GET|HEAD `/api/v1/jump/<service>?url=/blah/blah2.mp3` – returns 302 redirect to destination server
* GET|HEAD `/<service>?url=/blah/blah2.mp3` – same as above

## Config file format

```yaml
# top level services
service1:
    n1: # node id
        server: http://n1.radio-t.com     # base url 
        ping: /rtfiles/rt_podcast480.mp3  # ping url to check node's health
        method: HEAD                      # ping method, uses HEAD if nothing defined
        weight: 1                         # relative weight of the node [1..n]   

    n2:
        server: http://n2.radio-t.com
        ping: /rtfiles/rt_podcast480.mp3
        method: HEAD
        weight: 1

    n3:
        server: http://n3.radio-t.com
        ping: /rtfiles/rt_podcast480.mp3
        method: HEAD
        weight: 5                         # this node will get 5x hits comparing to n1 and n2 

service2:
    n1:
        server: http://n1.radio-t.com
        ping: /rtfiles/rt_podcast480.mp3
        method: GET
        weight: 1

    n2:
        server: http://n2.radio-t.com
        ping: /rtfiles/rt_podcast480.mp3
        method: GET
        weight: 3

    n3:
        server: http://n3.radio-t.com
        ping: /rtfiles/rt_podcast480.mp3
        method: GET
        weight: 1
```

## Stats

RLB does not implement any statistics internally but supports external service for requests like this:

```go
	type LogRecord struct {
		ID       string    `json:"id,omitempty"` // uniuque id
		FromIP   string    `json:"from_ip"`      // source ip
		TS       time.Time `json:"ts,omitempty"` // timestamp
		Fname    string    `json:"file_name"`    // requested file name
		Servcie  string	   `json:"service"`      // requested service
		DestHost string    `json:"dest"`         // picked destination node
	}
```

If stats url defined in command line or environment, each redirect will also hit stats url (`POST`) with `LogRecord` in body.
 
## Parameters

```
Usage:
  main [OPTIONS]

Application Options:
  -p, --port=    port (default: 7070) [$PORT]
  -c, --conf=    configuration file (default: rlb.yml) [$CONF]
  -r, --refresh= refresh interval (default: 30) [$REFRESH]
  -t, --timeout= HEAD/GET timeouts (default: 5) [$TIMEOUT]
  -s, --stats=   stats url [$STATS]
      --dbg      debug mode [$DEBUG]

```

## Status

RLB runs in production for several years and serves downloads from [radio-t](https://radio-t.com)
