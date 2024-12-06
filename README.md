# frp Operator

The frp Kubernetes Operator is a powerful tool that manages and automates the deployment of frp clients (connected to exit servers) and tunnels within your Kubernetes cluster. This operator watches two custom resources: `ExitServer` and `Tunnel`.

## What is frp?

[frp (Fast Reverse Proxy)](https://github.com/fatedier/frp) is an open-source tool that allows you to expose local servers to the internet securely and efficiently. It supports protocols like TCP, UDP, HTTP, and HTTPS, making it ideal for scenarios such as remote access, self-hosted applications, and cloud-native workloads.

## Generic Description

The frp Kubernetes Operator simplifies the deployment of frp clients and tunnels in your Kubernetes cluster. It eliminates the need for manual tunnels to ingress and external load balancer services.

### Use Cases

- Private cloud or home lab environments
- Self-hosted applications and APIs
- Testing and collaboration with colleagues or clients
- Integration with webhooks and third-party APIs

### Features

- No need for firewall port opening or port-forwarding rules
- Public IP is automatically assigned for TCP traffic
- Exit servers are created in your preferred cloud with cost-effective plans
- Compatible with any IngressController
- Portable IP address for flexibility

## Installation via Helm

To install the frp Operator using Helm, follow these steps:

1. Add the frp Helm repository:

    ```bash
    helm repo add frp https://frp-operator.aureum.cloud
    ```

2. Install the frp Operator:

    ```bash
    helm install my-frp-operator frp/frp-operator --version 1.0.0
    ```

   This will deploy the frp operator into your Kubernetes cluster.

## Exit Server Resource

An `ExitServer` resource defines the configuration for a frp exit server. Here's a sample ExitServer manifest:

```yaml
apiVersion: frp.aureum.cloud/v1
kind: ExitServer
metadata:
  name: exit-server-sample
spec:
  host: 12.345.67.89
  port: 7000
  authentication:
    token:
      secretKeyRef:
        name: exit-server-authentication
        key: token
```

### Add a secret

To add a secret for exit server authentication, you can use the following `kubectl` command:

```bash
kubectl -n sample create secret generic exit-server-authentication --from-literal=token=RDqQD6QX0ivEh2OGxjtagxpdQQoqYcAes5GrL0Wvp1XgsTE_FW
```

This command creates a generic secret named `exit-server-authentication` in the `sample` namespace. The secret contains a single key-value pair with the key `token` and the provided value `RDqQD6QX0ivEh2OGxjtagxpdQQoqYcAes5GrL0Wvp1XgsTE_FW`. This secret can then be referenced in your `ExitServer` resource for authentication.

### Exit Server Configuration

- **host**: IP address or domain of the exit server.
- **port**: Port on which the exit server is running.
- **authentication**: Configuration for server authentication.
  - **token**: Secret reference for authentication.

## Tunnel Resource

A `Tunnel` resource configures a frp tunnel to expose a port from a Kubernetes service. Below is a sample Tunnel manifest:

```yaml
apiVersion: frp.aureum.cloud/v1
kind: Tunnel
metadata:
  name: tunnel-sample
spec:
  exitServer: exit-server-sample
  tcp:
    localPort: 1234
    remotePort: 1234
    serviceRef:
      name: my-tcp-svc
      namespace: web-app
  transport:
    useEncryption: true
    useCompression: false
    proxyProtocol: v2
    bandwidthLimit: 100MB
```

### Tunnel Configuration

- **exitServer**: Reference to the associated ExitServer.
- **tcp**: TCP configuration for the tunnel.
  - **localPort**: Local port to expose.
  - **remotePort**: Remote port on the exit server.
  - **serviceRef**: Reference to the Kubernetes service.
- **transport**: Transport configuration for the tunnel.
  - **useEncryption**: Enable or disable encryption.
  - **useCompression**: Enable or disable compression.
  - **proxyProtocol**: Proxy protocol version.
  - **bandwidthLimit**: Bandwidth limit for the tunnel.

## Commands

### Get Exit Servers

```bash
$ kubectl get exitservers -A
```

Output:

```text
NAMESPACE   NAME                 HOST           PORT   SECRET
sample      exit-server-sample   12.345.67.89   7000   exit-server-authentication
```

### Get Tunnels

```bash
$ kubectl get tunnels -A
```

Output:

```text
NAMESPACE   NAME            EXIT SERVER          SERVICE NAMESPACE   SERVICE      LOCAL PORT   REMOTE PORT
sample      tunnel-sample   exit-server-sample   web-app             my-tcp-svc   1234         1234
```

### Get Tunnels with Additional Details

```bash
$ kubectl get tunnels -A -o wide
```

Output:

```text
NAMESPACE   NAME            EXIT SERVER          SERVICE NAMESPACE   SERVICE      LOCAL PORT   REMOTE PORT   ENCRYPTION   COMPRESSION   PROXY PROTOCOL   BANDWIDTH LIMIT
sample      tunnel-sample   exit-server-sample   web-app             my-tcp-svc   1234         1234          true         false         v2               100MB
```

## License

Copyright 2024 Aureum Cloud, N-Bit, Niek Berenschot.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
