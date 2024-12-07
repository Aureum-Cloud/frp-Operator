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

## Configuration Parameters

| Parameter                 | Description                        | Default                                |
|---------------------------|------------------------------------|----------------------------------------|
| `replicaCount`            | Number of replicas to deploy.     | `1`                                    |
| `image.repository`        | Container image repository.       | `ghcr.io/aureum-cloud/frp-operator`    |
| `image.pullPolicy`        | Image pull policy.                | `Always`                               |
| `image.tag`               | Container image tag.              | `""` (default tag)                     |
| `imagePullSecrets`        | List of image pull secrets.       | `[]`                                   |
| `nameOverride`            | Override the chart name.          | `""`                                   |
| `fullnameOverride`        | Override the full chart name.     | `""`                                   |
| `serviceAccount.create`   | Whether to create a service account. | `true`                              |
| `serviceAccount.name`     | Name of the service account.      | `controller-manager`                   |
| `podAnnotations`          | Annotations for pods.             | `{}`                                   |
| `podLabels`               | Labels for pods.                  | `{}`                                   |
| `podSecurityContext.runAsNonRoot` | Run pods as non-root user.      | `true`                              |
| `securityContext.allowPrivilegeEscalation` | Prevent privilege escalation. | `false`                          |
| `securityContext.capabilities.drop` | Capabilities to drop.         | `["ALL"]`                         |
| `service.type`            | Service type.                     | `ClusterIP`                            |
| `service.port`            | Service port.                     | `8443`                                 |
| `resources.limits.cpu`    | CPU limit.                        | `500m`                                 |
| `resources.limits.memory` | Memory limit.                     | `128Mi`                                |
| `resources.requests.cpu`  | CPU request.                      | `10m`                                  |
| `resources.requests.memory` | Memory request.                 | `64Mi`                                 |
| `nodeSelector`            | Node selector for pods.           | `{}`                                   |
| `tolerations`             | Tolerations for pod scheduling.   | `[]`                                   |
| `affinity`                | Affinity rules for pod scheduling.| `{}`                                   |

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
