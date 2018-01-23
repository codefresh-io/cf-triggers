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
    | | trigger:{event-uri} +-----> {pipeline-uid}  | |
    | |                     |     |                 | |
    | +---------------------+     | ...             | |
    |                             |                 | |
    |                             | {pipeline-uid}  | |
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
    | | pipeline:{pipeline-uid} +------> {event-uri} |  |
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

## Pipeline UID

Codefresh **Pipeline UID** is a unique identifier for Codefresh pipeline, as recognized by Codefresh.