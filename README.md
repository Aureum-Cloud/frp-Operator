# frp Operator Helm Chart

The **frp Kubernetes Operator** automates the deployment of FRP clients (connected to exit servers) and tunnels within your Kubernetes cluster. It watches two custom resources: `ExitServer` and `Tunnel`.

## Installation

1. **Add the Helm repository:**

   ```bash
   helm repo add frp https://frp-operator.aureum.cloud
   ```

2. **Install the FRP Operator:**

   ```bash
   helm install my-frp-operator frp/frp-operator --version 1.0.0
   ```

## frp Overview

[frp](https://github.com/fatedier/frp) is an open-source reverse proxy that allows you to expose local servers to the internet securely. It supports TCP, UDP, HTTP, and HTTPS.

## Resources

### ExitServer

Defines a FRP exit server configuration.

Example manifest:

```yaml
apiVersion: frp.aureum.cloud/v1
kind: ExitServer
metadata:
  name: exit-server
spec:
  host: 12.345.67.89
  port: 7000
  authentication:
    token:
      secretKeyRef:
        name: exit-server-auth
        key: token
```

### Tunnel

Defines a FRP tunnel to expose a service inside your cluster.

Example manifest:

```yaml
apiVersion: frp.aureum.cloud/v1
kind: Tunnel
metadata:
  name: tunnel
spec:
  exitServer: exit-server
  tcp:
    localPort: 1234
    remotePort: 1234
    serviceRef:
      name: my-service
      namespace: default
```

## Commands

- **Get ExitServers:**

   ```bash
   kubectl get exitservers -A
   ```

- **Get Tunnels:**

   ```bash
   kubectl get tunnels -A
   ```
