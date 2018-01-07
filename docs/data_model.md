# Redis Data Model

```text

    Secrets (String)

    +------------------------------------------+
    |                                          |
    |                                          |
    | +--------------------+      +----------+ |
    | |                    |      |          | |
    | | secret:{event-uri} +------> {secret} | |
    | |                    |      |          | |
    | +--------------------+      +----------+ |
    |                                          |
    |                                          |
    +------------------------------------------+


                Triggers (Sorted Set)

    +-------------------------------------------------+
    |                                                 |
    |                                                 |
    | +---------------------+     +-----------------+ |
    | |                     |     |                 | |
    | | trigger:{event-uri} +-----> {pipeline-uri}  | |
    | |                     |     |                 | |
    | +---------------------+     | ...             | |
    |                             |                 | |
    |                             | {pipeline-uri}  | |
    |                             |                 | |
    |                             +-----------------+ |
    |                                                 |
    |                                                 |
    +-------------------------------------------------+


                Pipelines (Sorted Set)

    +---------------------------------------------------+
    |                                                   |
    |                                                   |
    | +-------------------------+      +-------------+  |
    | |                         |      |             |  |
    | | pipeline:{pipeline-uri} +------> {event-uri} |  |
    | |                         |      |             |  |
    | +-------------------------+      | ...         |  |
    |                                  |             |  |
    |                                  | {event-uri} |  |
    |                                  |             |  |
    |                                  +-------------+  |
    |                                                   |
    |                                                   |
    +---------------------------------------------------+

```

## Event URI

**Event URI** is a unique identifier for trigger event. The exact event format is defined by *Event Provider*.

### Examples

```sh
# DockerHub
"index.docker.io:codefresh:cfapi:push"

# Cron
"cron:30 13 * * *"
```

## Pipeline URI

Codefresh **Pipeline URI** is a unique identifier for Codefresh pipeline.

```sh
# {account}:{repo-owner}:{repo-name}:{name}
"codefresh:codefresh-io:trigger-examples:run_pipeline"
```