# xweb

xweb allows Golang HTTP handlers to be registered with `bindings` that allow each API to be exposed via YAML configuration. The configuration can expose each API multiple times over different ports, network interfaces, and with the same root identity (x509 certificates) or separate ones. xweb allows Web APIs to be written once and then composed via configuratoin for exposure on publicly reachable network interfaces or not depending on the deployment model desired.

A great use case is development environments where all the APIs would be exposed on a single host/container vs a production environment that would expose each API differently based on sensitivity/security/access.

# More to Come!
