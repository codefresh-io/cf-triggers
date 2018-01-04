# Event Handler Contract

Any **Event Handler** should follow simple contract that consists from 2 sections: configuration and REST API.

## Configuration Contract

### JSON format

```json
{
  "type": "registry",
  "kind": "dockerhub",
  "service-url": "http://nomios:8080",
  "uri-template": "index.docker.io:{{repo-owner}}:{{repo-name}}:push",
  "uri-regex": "^index\\.docker\\.io:[a-z0-9_-]+:[a-z0-9_-]+:push$",
  "config": [
    {
      "name": "repo-owner",
      "type": "string",
      "validator": "^[A-z0-9]+$",
      "required": true
    },
    {
      "name": "repo-name",
      "type": "string",
      "validator": "^[A-z0-9]+$",
      "required": true
    }
  ]
}
```

### YAML format

```yaml
---
type: registry
kind: dockerhub
service-url: http://nomios:8080
uri-template: index.docker.io:{{repo-owner}}:{{repo-name}}:push
uri-regex: "^index\\.docker\\.io:[a-z0-9_-]+:[a-z0-9_-]+:push$"
config:
- name: repo-owner
  type: string
  validator: "^[A-z0-9]+$"
  required: true
- name: repo-name
  type: string
  validator: "^[A-z0-9]+$"
  required: true

```


- `type` - event type; e.g. `registry`, `cron`, `git`
- `kind` - (optional) event kind; e.g. `dockerhub`, `ecr`, `gcr`
- `service-url` - event handler service url (including protocol and port); `hermes` invokes `REST API`
- `uri-template` - *mustache template* for `event URI`
- `uri-regex` - `event URI` regex; used for validation and matching
- `config` - configuration of template parameters (`array`)
- - `name` - parameter name; **same** as in *template*
- - `type` - parameter type: `string`, `date`, `bool`, `int`, etc.
- - `validator` - parameter value validator: can be `regexp`, range (for `date` and `int`), enum list, etc.
- - `required` - is it a required parameter; non-required parameter uses some `default` value

### Configuration Contract Discovery

An **Event Handler** configuration contract is discovered automatically on Kubernetes cluster. To support configuration discovery an **Event Handler** should save *configuration contract* as *ConfigMap* and label this config map with `config: event-handler` *Label*.

It's recommended (not must) to add additional labels:

- `type: <Event Handler Type>`
- `kind: <Event Handler Kind>`

## REST API

### Get Event Information

  Returns extended event information, given `event-uri`.

#### URL

```text
/event-info/:uri
```

#### Method

```text
GET
```

#### URL Params

**Required:**

```text
uri=[string]

# example: `/event-info/index.docker.io:codefresh:fortune:push
```

##### Success Response

- **Code:** `200`
    **Content:**
    ```json
    {
        "endpoint": "https://g.codefresh.io/dockerhub?secret=64zy952f3",
        "description": "DockerHub codefresh/fortune push event",
        "status": "active"
    }
    ```

- `endpoint` - public address (or IP) of endpoint API; usually used as a webhook endpoint
- `description` - human-readable "translation" of `event-uri`
- `status` - event handler status, usually `active` or `non-active` (can be other)

##### Error Response

- **Code:** `404 NOT FOUND`
