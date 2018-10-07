# RLB - Redirecting Load Balancer [![Build Status](https://travis-ci.org/umputun/rlb.svg?branch=master)](https://travis-ci.org/umputun/rlb) [![Docker Automated build](https://img.shields.io/docker/automated/jrottenberg/ffmpeg.svg)](https://hub.docker.com/r/umputun/rlb/)

This service redirects incoming `GET` and `HEAD` requests to the upstream servers. 
Servers picked up randomly, unhealthy boxes excluded dynamically.

## API

* GET|HEAD `/api/v1/jump/<service>?url=/blah/blah2.mp3`

## Config file format
```
service1:
    n1:
        server: http://n1.radio-t.com
        ping: /rtfiles/rt_podcast480.mp3
        method: HEAD
        weight: 1

    n2:
        server: http://n2.radio-t.com
        ping: /rtfiles/rt_podcast480.mp3
        method: HEAD
        weight: 1

    n3:
        server: http://n3.radio-t.com
        ping: /rtfiles/rt_podcast480.mp3
        method: HEAD
        weight: 5

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

## Parameters

```
Usage:
  main [OPTIONS]

Application Options:
  -c, --conf=    configuration file (default: rlb.yml) [$CONF]
  -r, --refresh= refresh interval (secs) (default: 30) [$REFRESH]
  -t, --timeout= HEAD/GET timeout (secs) (default: 5) [$TIMEOUT]
```

## Status

RLB in production for several years to serve downloads from [radio-t](https:/radio-t.com)
