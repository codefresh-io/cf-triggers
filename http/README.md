# REST API

Invoke `hermes` REST API through `cfapi` API proxy.

If you are using Visual Studio Code, install [REST Client](https://marketplace.visualstudio.com/items?itemName=humao.rest-client) plugin.

Add a new REST Client **Environment** with `codefresh-url` and `api-token` (see plugin documentation).

```json
"rest-client.environmentVariables": {
    "$shared": {},
    "production": {
        "codefresh-url": "https://g.codefresh.io",
        "api-token": "XXXXXXXX"
    },
    "staging": {
        "codefresh-url": "https://app-staging.codefresh.io",
        "api-token": "YYYYYYYY"
    }
}
```