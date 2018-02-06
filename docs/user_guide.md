# User Guide

> Disclaimer: this is a temporary user guide. Once we configure API Gateway, we will allow managing triggers through official `codefresh` cli and UI

## Accessing Hermes command line client

`hermes` works both as trigger manager server and as command line client for managing triggers.

Find `hermes` pod running on Kubernetes cluster.

```sh
export HERMES_POD=$(kubectl get pod -l app=hermes -o jsonpath="{.items[0].metadata.name}")
```

From now on you can work either with `hermes` REST API or using command line.

For API do a port forwarding:

```sh
kubectl port-forward ${HERMES_POD} 8080 &
```

For command line tool get pod shell:

```sh
kubectl exec -it ${HERMES_POD} sh
```

### Adding new Docker Hub push trigger event

It's possible to trigger Codefresh pipeline when Docker image pushed to Docker Hub.
Select Docker Hub repository, that will trigger Codefresh pipeline and **construct** trigger `event URI` following pattern (matching `regex`) bellow:

```sh
# Pattern
# index.docker.io:{{Docker Hub Account}}:{{Docker Hub Repo}}:push
```

Regex:

```regexp
^index\.docker\.io:[a-z0-9_-]+:[a-z0-9_-]+:push$
```

#### Example

Adding trigger for `codefresh/fortune` Docker Hub `push` event, that should trigger Codefresh pipeline  `run_fortune` execution for `alexei-led` account `codefresh-io/trigger-examples`.

#### command line

TODO: show command
```sh

```

```sh
# add new trigger (and generate secret)
TODO: example
```

#### REST API

```sh

###
# Add trigger
TODO: update
POST http://localhost:8080/triggers
content-type: application/json

{
    "event": "index.docker.io:codefresh:fortune:push",
    "secret": "!generate",
    "pipelines": [
        "5a439664af73ad0001f3ece0",
        "89893364af73ad0321f3abce"
    ]
}
```

### Getting event details

#### get event info - command line

```sh
NAME:
   hermes info event - get event details

USAGE:
   hermes info event <event URI>

DESCRIPTION:
   Get event details from event uri

```

```sh
hermes info event index.docker.io:codefresh:fortune:push
```

```text
# Output

endpoint: https://g.codefresh.io/nomios/dockerhub?secret=icnuv8JJXGwx1CzU
description: Docker Hub codefresh/fortune push event
status: active
help: |-
  Docker Hub webhooks fire when an image is built in, pushed or a new tag is added to, your repository.

  Configure Docker Hub webhooks on https://hub.docker.com/r/codefresh/fortune/~/settings/webhooks/

  Add following Codefresh Docker Hub webhook endpoint https://g.codefresh.io/nomios/dockerhub?secret=icnuv8JJXGwx1CzU

```

Follow the above guide and set Docker Hub webhook.

#### get event info - REST API

```sh
###
# DEBUG kubectl port-forward <hermes-pod> 8080 &
# Get Event Info details for DockerHub codefresh/fortune push event
# NOTE: create trigger pipeline before
GET http://localhost:8080/events/info/index.docker.io:codefresh:fortune:push
```

You should see output similar to one from command line tool, but in `JSON` format.

### Next

There are other trigger management commands available both for `hermes` command line tool and REST API.

Run `hermes --help` to see other command, or inspect `server.go` code to see REST API endpoints.