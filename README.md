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
   - STORE_HOST         - set the url to the Redis store server (default localhost)
   - STORE_PORT         - set Redis store port (default to 6379)
   - STORE_PASSWORD     - set Redis store password

Copyright © Codefresh.io

VERSION:
   0.5.2
  git-commit: 6f88d67
  build-date: 2018-01-11_14:43_GMT
  platform: darwin amd64 go1.9.2

AUTHOR:
   Alexei Ledenev <alexei@codefresh.io>

COMMANDS:
     server    start trigger manager server
     trigger   configure Codefresh triggers
     pipeline  configure Codefresh trigger pipelines
     info      get information about installed event handlers and events
     help, h   Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --codefresh value, -c value       Codefresh API endpoint (default: "https://g.codefresh.io/") [$CFAPI_URL]
   --token value, -t value           Codefresh API token [$CFAPI_TOKEN]
   --redis value, -r value           redis store host name (default: "localhost") [$STORE_HOST]
   --redis-port value, -p value      redis store port (default: 6379) [$STORE_PORT]
   --redis-password value, -s value  redis store password [$STORE_PASSWORD]
   --config value                    type config file (default: "/etc/hermes/type_config.json") [$TYPES_CONFIG]
   --skip-monitor, -m                skip monitoring config file for changes
   --log-level value, -l value       set log level (debug, info, warning(*), error, fatal, panic) (default: "warning") [$LOG_LEVEL]
   --dry-run, -x                     do not execute commands, just log
   --json, -j                        produce log in JSON format: Logstash and Splunk friendly
   --help, -h                        show help
   --version, -v                     print the version

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
