# Hermes - Codefresh Trigger Manager

[![Codefresh build status](https://g.codefresh.io/api/badges/build?repoOwner=codefresh-io&repoName=hermes&branch=master&pipelineName=hermes&accountName=codefresh-inc&type=cf-1)](https://g.codefresh.io/repositories/codefresh-io/hermes/builds?filter=trigger:build;branch:master;service:5a2f9f604a678d0001da7621~hermes) [![Go Report Card](https://goreportcard.com/badge/github.com/codefresh-io/hermes)](https://goreportcard.com/report/github.com/codefresh-io/hermes) [![codecov](https://codecov.io/gh/codefresh-io/hermes/branch/master/graph/badge.svg)](https://codecov.io/gh/codefresh-io/hermes)

[![](https://images.microbadger.com/badges/image/codefresh/hermes.svg)](http://microbadger.com/images/codefresh/hermes) [![](https://images.microbadger.com/badges/commit/codefresh/hermes.svg)](https://microbadger.com/images/codefresh/hermes) [![Docker badge](https://img.shields.io/docker/pulls/codefresh/hermes.svg)](https://hub.docker.com/r/codefresh/hermes/)

Codefresh Trigger Manager (aka `hermes`) is responsible for processing *normalized events* coming from different *Event Providers* and triggering Codefresh pipeline execution using variables extracted from *events* payload.

## Normalized Event

It's responsibility of *Event Provider* to get interesting events (or generate; `cron` for example) from external system either with WebHook or using some pooling technique, extract *unique event URI* and *normalize* these events and send then to `hermes`.

### Normalization format

```json
{
    "secret": "secret (or payload signature)",
    "variables": {
        "key1": "value",
        "key2": "value2",
        "...": "...",
        "keyN": "valueN"
    },
    "original" : "base64enc(original.payload)"
}
```

- `secret` - validation secret or `hmac` signature (`sha1`, `sha256`, `sha512`); webhook payload `hmac` signature for example
- `variables` - list of *selected* event properties, extracted from event payload
- `original` - original event payload (JSON or FORM), `base64` encoded

### Event unique URI

> **Claim** every event has an URI, that can be easily constructed!

Based on above **claim**, we can construct unique URI for any event coming from external system.

#### Examples

| External System         | Event Description                                             | Event URI                              |
| ----------------------- | ------------------------------------------------------------- | -------------------------------------- |
| DockerHub               | push `cfapi` docker image with new tag                        | `index.docker.io:codefresh:cfapi:push` |
| GitHub                  | publish new GitHub release for `pumba`                        | `github.io:gaiaadm:pumba:release`      |
| TravisCI                | completed TravisCI build for `alexei-led/alpine-plus`         | `travis-ci.org:alexei-led:alpine-plus` |
| Cron                    | once a Day, at 1:30pm: `30 13 * * *`                          | `cron:30 13 * * *`                     |
| Private Docker Registry | push `demo\demochat` to private Docker registry `myhost:5000` | `registry:myhost:5000:demo:demochat`   |

## Trigger Manager

Hermes trigger manager is a single binary file `hermes`. This file includes both configuration CLI and trigger manager server.

```text
NAME:
   hermes - configure triggers and run trigger manager server

USAGE:
   Configure triggers for Codefresh pipeline execution or start trigger manager server. Process "normalized" events and run Codefresh pipelines with variables extracted from events payload.

    ╦ ╦┌─┐┬─┐┌┬┐┌─┐┌─┐  ╔═╗┌─┐┌┬┐┌─┐┌─┐┬─┐┌─┐┌─┐┬ ┬  ╔╦╗┬─┐┬┌─┐┌─┐┌─┐┬─┐┌─┐
    ╠═╣├┤ ├┬┘│││├┤ └─┐  ║  │ │ ││├┤ ├┤ ├┬┘├┤ └─┐├─┤   ║ ├┬┘││ ┬│ ┬├┤ ├┬┘└─┐
    ╩ ╩└─┘┴└─┴ ┴└─┘└─┘  ╚═╝└─┘─┴┘└─┘└  ┴└─└─┘└─┘┴ ┴   ╩ ┴└─┴└─┘└─┘└─┘┴└─└─┘

hermes respects following environment variables:
   - REDIS_HOST         - set the url to the Redis server (default localhost)
   - REDIS_PORT         - set Redis port (default to 6379)
   - REDIS_PASSWORD     - set Redis password

Copyright © Codefresh.io

VERSION:
   Codefresh Hermes 0.2.6
  git-commit: 8a579f0
  build-date: 2017-12-19_14:22_GMT
  platform: darwin amd64 go1.9.2

AUTHOR:
   Alexei Ledenev <alexei@codefresh.io>

COMMANDS:
     server    start trigger manager server
     trigger   configure Codefresh triggers
     pipeline  manage Codefresh trigger pipelines
     help, h   Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --codefresh value, --cf value  Codefresh API endpoint (default: "https://g.codefresh.io/") [$CFAPI_URL]
   --token value, -t value        Codefresh API token [$CFAPI_TOKEN]
   --redis value                  redis host name (default: "localhost") [$REDIS_HOST]
   --redis-port value             redis host port (default: 6379) [$REDIS_PORT]
   --redis-password value         redis password [$REDIS_PASSWORD]
   --debug                        enable debug mode with verbose logging [$DEBUG_HERMES]
   --dry-run                      do not execute commands, just log
   --json                         produce log in JSON format: Logstash and Splunk friendly
   --help, -h                     show help
   --version, -v                  print the version
```

## Deploy with Helm

*Hermes* uses Codefresh API to execute pipelines and requires to pass non-expiring API token for working installation.
When you install *Hermes* as a separate release (from *Codefresh*), you must also pass a `CFAPI_URL`.

```sh
helm install hermes --set cfapi.token="GET-CFAPI-TOKEN"
```

## Building Hermes

`hermes` requires Go SDK to build.

1. Clone this repository into `$GOPATH/src/github.com/codefresh-io/hermes`
1. Run `hack/build.sh` helper script or `go build cmd/main.go`
1. Run `hack/test.sh` helper script to test
