# xweb

xweb allows Go HTTP handlers to be registered with `bindings` that let each API be exposed via YAML
configuration. A single codebase can expose APIs on multiple ports, network interfaces, and with
independent TLS identities — all without code changes. The composition is entirely driven by config.

A common use case: development exposes every API on one port; production splits them across different
interfaces and certificates depending on sensitivity and access requirements.

## Concepts

| Concept                      | Description                                                                                   |
|------------------------------|-----------------------------------------------------------------------------------------------|
| **Instance**                 | The root xweb object. Owns the registry, config, and all servers.                            |
| **Registry**                 | Maps binding strings to `ApiHandlerFactory` implementations.                                 |
| **ServerConfig**             | One logical server — a name, a set of APIs, a set of bind points, and optional TLS identity. |
| **ApiHandlerFactory**        | Creates `ApiHandler` instances for a given binding. Registered once at startup.              |
| **ApiHandler**               | An `http.Handler` with a binding name, root path, and routing predicate (`IsHandler`).       |
| **BindPoint**                | Where a server listens — either a TCP underlay address or an OpenZiti overlay service.       |
| **BindPointListenerFactory** | Creates `BindPoint` instances from configuration.                                            |

## Configuration

xweb reads from a YAML config map. The default section names are `identity` (root TLS identity) and
`web` (array of server definitions), though both are configurable.

```yaml
identity:
  cert:        /path/to/cert.pem
  server_cert: /path/to/server-cert.pem
  key:         /path/to/key.pem
  ca:          /path/to/ca-chain.pem

web:
  - name: public-apis
    bindPoints:
      - interface: 0.0.0.0:443
        address:   public.example.com:443
    apis:
      - binding: my-api
    options:
      minTLSVersion: TLS1.2
      maxTLSVersion: TLS1.3
      readTimeout:   5s
      writeTimeout:  10s
      idleTimeout:   5s
```

Multiple servers can be defined, each with their own bind points, APIs, and optional identity
override:

```yaml
web:
  - name: external
    bindPoints:
      - interface: 0.0.0.0:443
        address:   external.example.com:443
    apis:
      - binding: client-api

  - name: internal
    identity:
      cert: /path/to/internal-cert.pem
      key:  /path/to/internal-key.pem
      ca:   /path/to/internal-ca.pem
    bindPoints:
      - interface: 127.0.0.1:1280
        address:   127.0.0.1:1280
    apis:
      - binding: management-api
      - binding: health-check
```

### Bind Points

A **bind point** defines both where the server listens (`interface`) and what address it advertises
to clients (`address`). Multiple bind point types are supported and a server may have more than one.

#### Underlay (TCP)

Standard TCP listener. `interface` is the local `host:port` to bind; `address` is the
publicly-reachable `host:port` clients should use.

```yaml
bindPoints:
  - interface: 0.0.0.0:443
    address:   myhost.example.com:443
```

#### Overlay (OpenZiti service)

Listens on an OpenZiti service instead of a TCP port. The server is reachable only through the
OpenZiti network. Overlay bind points have no conventional `host:port` and are omitted from
advertised `apiBaseUrls`.

```yaml
bindPoints:
  - identity:
      file:    /path/to/identity.json
      service: my-ctrl-service
      tlsClientAuthenticationPolicy: RequireAndVerifyClientCert  # optional
```

### API Options

Each entry in `apis` has a required `binding` and an optional `options` map whose keys are
interpreted by the `ApiHandlerFactory` for that binding:

```yaml
apis:
  - binding: my-api
    options:
      someKey: someValue
```

### Server Options

| Option          | Default | Description                                                           |
|-----------------|---------|-----------------------------------------------------------------------|
| `minTLSVersion` | TLS1.2  | Minimum TLS version accepted by the server.                           |
| `maxTLSVersion` | TLS1.3  | Maximum TLS version accepted by the server.                           |
| `readTimeout`   | 5s      | Maximum duration to read the full request.                            |
| `writeTimeout`  | 10s     | Maximum duration to write the full response.                          |
| `idleTimeout`   | 5s      | Maximum time to wait for the next request on a keep-alive connection. |

Valid TLS version values:

| Value    | Protocol |
|----------|----------|
| `TLS1.0` | TLS 1.0  |
| `TLS1.1` | TLS 1.1  |
| `TLS1.2` | TLS 1.2  |
| `TLS1.3` | TLS 1.3  |

## Go Usage

### 1. Implement ApiHandlerFactory and ApiHandler

```go
type MyApiHandler struct {
    mux *http.ServeMux
}

func (h *MyApiHandler) Binding() string                      { return "my-api" }
func (h *MyApiHandler) Options() map[interface{}]interface{} { return nil }
func (h *MyApiHandler) RootPath() string                     { return "/my/api" }
func (h *MyApiHandler) IsHandler(r *http.Request) bool       { return strings.HasPrefix(r.URL.Path, h.RootPath()) }
func (h *MyApiHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    h.mux.ServeHTTP(w, r)
}

type MyApiFactory struct{}

func (f *MyApiFactory) Binding() string { return "my-api" }

func (f *MyApiFactory) New(serverConfig *xweb.ServerConfig, options map[interface{}]interface{}) (xweb.ApiHandler, error) {
    mux := http.NewServeMux()
    mux.HandleFunc("/my/api/hello", func(w http.ResponseWriter, r *http.Request) {
        _, _ = w.Write([]byte("hello"))
    })
    return &MyApiHandler{mux: mux}, nil
}

func (f *MyApiFactory) Validate(config *xweb.InstanceConfig) error { return nil }
```

### 2. Implement BindPointListenerFactory

xweb ships no built-in bind point implementations — consumers register their own via the global
`BindPointListenerFactoryRegistry`. See [openziti/ziti's `common/bindpoints`](https://github.com/openziti/ziti)
for a reference implementation of both underlay (TCP) and overlay (OpenZiti) bind points.

```go
type MyBindPointFactory struct{}

func (f *MyBindPointFactory) New(conf map[interface{}]interface{}) (xweb.BindPoint, error) {
    // parse conf, return a BindPoint
}

// Register at startup, before LoadConfig is called:
xweb.BindPointListenerFactoryRegistry = append(xweb.BindPointListenerFactoryRegistry, &MyBindPointFactory{})
```

A `BindPoint` must implement:

```go
type BindPoint interface {
    Listener(serverName string, tlsConfig *tls.Config) (net.Listener, error)
    BeforeHandler(next http.Handler) http.Handler
    AfterHandler(prev http.Handler) http.Handler
    Validate(identity.Identity) error
    ServerAddress() string
    Type() BindPointType
}
```

`Type()` returns a `BindPointType` string that callers can use to distinguish bind point kinds —
for example, to skip overlay bind points when building advertised URL lists.

### 3. Create and start an Instance

```go
// Build registry and register factories
registry := xweb.NewRegistryMap()
_ = registry.Add(&MyApiFactory{})

// Create instance using a pre-loaded identity
instance := xweb.NewDefaultInstance(registry, myIdentity)

// Parse and validate configuration from your config map
if err := instance.LoadConfig(cfgMap); err != nil {
    log.Fatalf("xweb config error: %v", err)
}

// Build servers then start them, or call Run() to do both
instance.Build()
instance.Start()
// or: instance.Run()

// Graceful shutdown
instance.Shutdown()
```

### 4. Access xweb context from a handler

Inside an `http.Handler`, retrieve the active `ServerContext` (which exposes the `InstanceConfig`,
`ServerConfig`, and bind points for the current request):

```go
func (h *MyApiHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    serverCtx := xweb.ServerContextFromRequestContext(r.Context())
    if serverCtx != nil {
        _ = serverCtx.Config // *xweb.InstanceConfig
    }
}
```

### 5. Filtering bind points by type

Use `BindPointType` to distinguish bind point kinds when iterating. For example, only underlay bind
points have a conventional `host:port` suitable for inclusion in advertised URL lists:

```go
for _, bp := range serverConfig.BindPoints {
    if bp.Type() != mypackage.BindPointTypeUnderlay {
        continue // skip overlay and any future bind point types
    }
    advertised = append(advertised, "https://"+bp.ServerAddress()+"/my/api")
}
```
