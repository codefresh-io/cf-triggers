# Event Provider Contract

Any Trigger **Event Provider** should follow simple contract that consists from 2 sections: configuration and REST API.

## Configuration Contract

### JSON format

```json
{
  "type": "registry",
  "kind": "dockerhub",
  "service-url": "http://nomios:8080",
  "uri-template": "registry:dockerhub:{{namespace}}:{{name}}:push",
  "uri-regex": "^registry:dockerhub:[a-z0-9_-]+:[a-z0-9_-]+:push$",
  "config": [
    {
      "name": "namespace",
      "type": "string",
      "validator": "^[a-z0-9_-]+$",
      "required": true
    },
    {
      "name": "name",
      "type": "string",
      "validator": "^[a-z0-9_-]+$",
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
uri-template: registry:dockerhub:{{namespace}}:{{name}}:push
uri-regex: "^registry:dockerhub:[a-z0-9_-]+:[a-z0-9_-]+:push$"
config:
- name: namespace
  type: string
  validator: "^[a-z0-9_-]+$"
  required: true
- name: name
  type: string
  validator: "^[a-z0-9_-]+$"
  required: true

```

- `type` - event type; e.g. `registry`, `cron`, `git`
- `kind` - (optional) event kind; e.g. `dockerhub`, `ecr`, `gcr`
- `service-url` - event provider service url (including protocol and port); `hermes` invokes `REST API`
- `uri-template` - *mustache template* for `event URI`
- `uri-regex` - `event URI` regex; used for validation and matching
- `config` - configuration of template parameters (`array`)
- - `name` - parameter name; **same** as in *template*
- - `type` - parameter type: `string`, `date`, `bool`, `int`, etc.
- - `validator` - parameter value validator: can be `regexp`, range (for `date` and `int`), enum list, etc.
- - `required` - is it a required parameter; non-required parameter uses some `default` value

### Configuration Contract Discovery

An **Event Handler** configuration contract is discovered automatically on Kubernetes cluster. To support configuration discovery an **Event Handler** should save *configuration contract* as *ConfigMap* and label this config map with `config: event-provider` *Label*.

It's recommended (not must) to add additional labels:

- `config: event-provider`
- `type: <Event Provider Type>`
- `kind: <Event Provider Kind>`

## REST API

---

### Get Event Information

> This is required

  Returns extended trigger event information, given `event-uri`.

#### URL

```text
/event/:uri/:secret
```

#### Method

```text
GET
```

#### URL Params

**Required:**

```text
uri=[string]

# example: `/event/registry:dockerhub:codefresh:fortune:push/64zy952f3
```

##### Success Response

- **Code:** `200`
    **Content:**
    ```json
    {
        "endpoint": "https://g.codefresh.io/dockerhub?secret=64zy952f3",
        "description": "DockerHub codefresh/fortune push event",
        "help": "Very long and helpful text (Markdown-formatted)",
        "status": "active"
    }
    ```

- `endpoint` - public address (or IP) of endpoint API; usually used as a webhook endpoint
- `description` - human-readable "translation" of `event-uri`
- `help` - very long and helpful text (markdown-formatted)
- `status` - event status from event provider, usually `active` or `non-active` (can be other)

##### Error Response

- **Code:** `404 NOT FOUND`

---

### Subscribe to Event

> This is optional method. Return 501 if not supported.

  Subscribe to event in external system (using available API, for example). Returns extended trigger event information for *subscribed* event.

#### URL

```text
/event/:uri/:secret/:credentials
```

#### Method

```text
POST
```

#### URL Params

**Required:**

```text
uri=[string]
secret=[string]
credentials=[base64(string)]

# example: `/event/registry:dockerhub:codefresh:fortune:push/64zy952f3/ewoidXNlciI6ICJhZG1pbiIsCiJwYXNzd29yZCI6ICJyb290Igp9Cg==
```

##### Success Response

- **Code:** `200`
    **Content:**
    ```json
    {
        "endpoint": "https://g.codefresh.io/dockerhub?secret=64zy952f3",
        "description": "DockerHub codefresh/fortune push event",
        "help": "Very long and helpful text (Markdown-formatted)",
        "status": "active"
    }
    ```

- `endpoint` - public address (or IP) of endpoint API; usually used as a webhook endpoint
- `description` - human-readable "translation" of `event-uri`
- `help` - very long and helpful text (markdown-formatted)
- `status` - event status from event provider, usually `active` or `non-active` (can be other)

##### Error Response

- **Code:** `401 Unauthorized` when wrong credentials are passed
- **Code:** `403 Forbidden` when no sufficient permissions
- **Code:** `404 Not Found` when event source is not found
- **Code:** `500 Internal Server Error` for any other error
- **Code:** `501 Not Implemented` method not implemented

---

### Unsubscribe from Event

> This is optional method. Return 501 if not supported.

  Unsubscribe from event in external system (using available API, for example).

#### URL

```text
/event/:uri/:credentials
```

#### Method

```text
DELETE
```

#### URL Params

**Required:**

```text
uri=[string]
credentials=[base64(string)]

# example: `/event/registry:dockerhub:codefresh:fortune:push/ewoidXNlciI6ICJhZG1pbiIsCiJwYXNzd29yZCI6ICJyb290Igp9Cg==
```

##### Success Response

- **Code:** `200` successfully unsubscribed

##### Error Response

- **Code:** `401 Unauthorized` when wrong credentials are passed
- **Code:** `403 Forbidden` when no sufficient permissions
- **Code:** `404 Not Found` when event source is not found
- **Code:** `500 Internal Server Error` for any other error
- **Code:** `501 Not Implemented` method not implemented
