# Hermes - Codefresh Trigger Manager

Codefresh Trigger Manager (aka `hermes`) is responsible for processing *normalized events* coming from different *Event Sources* and triggering Codefresh pipeline execution using variables extracted from *events* payload.

## Normalized Event

It's responsibility of *Event Source* to get interesting events (or generate; `cron` for example) from external system either with WebHook or using some pooling technique, extract *unique event URI* and *normalize* these events and send then to `hermes`.

### Normalization format

```json
{
    "secret": "very secret secret!",
    "variables": {
        "key1": "value",
        "key2": "value2",
        ...
        "keyN": "valueN"
    },
    "original" : "base64enc(original.payload)"
}
```

- `secret` - validation secret, can be used for trigger protection (as WebHook secret, for example)
- `variables` - list of *selected* event properties, extracted from event payload
- `original` - original event payload (JSON or FORM), `base64` encoded

### Event unique URI

> **Claim** every event has an URI, that can be easily constructed!

Based on above **claim**, we can construct unique URI for any event coming from external system.

#### Examples

| External System         | Event Description                                             | Event URI                              |
| ----------------------- | ------------------------------------------------------------- | -------------------------------------- |
| DockerHub               | push `cfapi` docker image with new tag                        | `index.docker.io:codefresh:cfapi`      |
| GitHub                  | publish new GitHub release for `pumba`                        | `github.io:gaiaadm:pumba:release`      |
| TravisCI                | completed TravisCI build for `alexei-led/alpine-plus`         | `travis-ci.org:alexei-led:alpine-plus` |
| Cron                    | once a Day, at 1:30pm: `30 13 * * *`                          | `cron:30 13 * * *`                     |
| Private Docker Registry | push `demo\demochat` to private Docker registry `myhost:5000` | `registry:myhost:5000:demo:demochat`   |

## Event Flow Diagram

```ascii
          DockerHub Event Source        hermes trigger manager       pipeline manager (cfapi)

                    +                            +                             +
original event      |                            |                             |
                    |                            |                             |
        +---------> |  normalized event (vars)   |                             |
                    |                            |                             |
                    | +------------------------> |     pipeline: p1(vars)      |
                    |                            |                             |
                    |                            | +-------------------------> |
                    |     OK: running p1,p2,p3   |                             |
                    | <------------------------+ |                             |
                    |                            |     pipeline: p2(vars)      |
                    |                            |                             |
                    |                            | +-------------------------> |
                    |                            |                             |
                    |                            |                             |
                    |                            |     pipeline: p3(vars)      |
                    |                            |                             |
                    |                            | +-------------------------> |
                    |                            |                             |
                    |                            |                             |
                    +                            +                             +

```

## Trigger Manager

Hermes trigger manager is a single binary file `hermes`. This file includes both configuration CLI and trigger manager server.

```
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
   0.2.0
  git-commit: 42279b2
  build-date: 2017-12-06_11:26_GMT
  platform: darwin amd64 go1.9.2
   
AUTHOR(S):
   Alexei Ledenev <alexei@codefresh.io> 
   
COMMANDS:
     server   start trigger manager server
     trigger  configure Codefresh triggers
     help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --redis value           redis host name (default: "localhost") [$REDIS_HOST]
   --redis-port value      redis host port (default: 6379) [$REDIS_PORT]
   --redis-password value  redis password [$REDIS_PASSWORD]
   --debug                 enable debug mode with verbose logging
   --dry-run               do not execute commands, just log
   --json                  produce log in JSON format: Logstash and Splunk friendly
   --help, -h              show help
   --version, -v           print the version
   

```

## Building Hermes

`hermes` requires Go SDK to build.

1. Clone this repository into `$GOPATH/src/github.com/codefresh-io/hermes`
1. Run `hack/build.sh` helper script or `go build cmd/main.go`