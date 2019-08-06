## Build status
[![Build Status](https://dev.azure.com/ktharchitecture/logspout-gelf-tls/_apis/build/status/mictsi.logspout-gelf-tls?branchName=master)](https://dev.azure.com/ktharchitecture/logspout-gelf-tls/_build/latest?definitionId=5&branchName=master)

# Logspout with GELF adapter and TCP+TLS support
A logspout docker image with TLS support. 

This image contains [Logspout](https://github.com/gliderlabs/logspout) which is compiled with [GELF adapter] and originally forked from https://github.com/karlvr/logspout-gelf so you can forward Docker logs in GELF to a Graylog server.

# Docker hub images
Images are published in the following registry, https://hub.docker.com/r/kthse/logspout-gelf-tls

## Usage
Modify the docker compose file to point to your graylogserver and run it with the docker-compose command.

Use  gelf://my.log.server:12201 as protocol for gelf over udp and  gelf+tls://my.log.server:12201 for gelf over TCP+TLS


### CLI example for gelf over udp
`docker run -d --name=logspout --restart=unless-stopped -h $(hostname -f) -v /var/run/docker.sock:/var/run/docker.sock kthse/logspout-gelf-tls:latest gelf://my.log.server:12201`

### CLI example for gelf over TCP and with TLS encryption
`docker run -d --name=logspout --restart=unless-stopped -h $(hostname -f) -v /var/run/docker.sock:/var/run/docker.sock kthse/logspout-gelf-tls:latest gelf+tls://my.log.server:12201`

### Docker Compose example
You could use this image with the following docker-compose file for unencrypted gelf messages over udp:

```
version: '3'

services:
  logspout:
    image: kthse/logspout-gelf-tls:108
    hostname: my.message.source
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    command: gelf://my.log.server:12201
    restart: unless-stopped
```

or the following for sending gelf message over TLS+TCP: 

```
version: '3'

services:
  logspout:
    image: kthse/logspout-gelf-tls:latest
    hostname: my.message.source
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    command: gelf+tls://my.log.server:12201
    restart: unless-stopped
```

## Disclaimer

This image is provided as-is and only with best effort. We try to update this image with the latest Logspout stable version.
