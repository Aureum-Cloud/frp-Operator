# Ingress Adapter Example

This guide explains how to use the frp-operator Ingress adapter to convert native Kubernetes `Ingress` resources into FRP `Tunnel` resources and expose Kubernetes Services through an `ExitServer`.

## How it works

1. The adapter only processes an Ingress whose `spec.ingressClassName` references an `IngressClass` with `spec.controller: frp.aureum.cloud/ingress-controller`.
2. The Ingress must specify an `ExitServer` in the same namespace through the `frp.aureum.cloud/exit-server` annotation.
3. The adapter creates one HTTP `Tunnel` for every host, path, and backend combination.
4. Generated Tunnels are owned by the Ingress. Updating or deleting the Ingress updates or removes its Tunnels.
5. Named Service ports are automatically resolved to numeric Service ports.

## Prerequisites

- frp-operator is installed in the Kubernetes cluster, including its CRDs and RBAC resources.
- A reachable frps server is available.
- The Kubernetes Service that you want to expose has been deployed.

## Step 1: Create the authentication Secret

Create a Secret containing the token configured on your frps server:

```bash
kubectl create secret generic exit-server-authentication \
  --from-literal=token='your-frps-token'
```

## Step 2: Create an ExitServer

Replace `host` and `port` with the address of your frps server:

```yaml
apiVersion: frp.aureum.cloud/v1
kind: ExitServer
metadata:
  name: exit-server-sample
spec:
  host: 12.23.34.45
  port: 7000
  authentication:
    token:
      secretKeyRef:
        name: exit-server-authentication
        key: token
```

Apply the resource:

```bash
kubectl apply -f exitserver.yaml
```

Verify that the ExitServer exists and its frpc Pod is running:

```bash
kubectl get exitservers
kubectl get pods -l app.kubernetes.io/created-by=exit-server-sample
```

## Step 3: Deploy an example application

Skip this step if you already have a Kubernetes Service to expose.

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: example
spec:
  replicas: 1
  selector:
    matchLabels:
      app: example
  template:
    metadata:
      labels:
        app: example
    spec:
      containers:
        - name: example
          image: hashicorp/http-echo
          args: ["-text=hello from example"]
          ports:
            - containerPort: 5678
---
apiVersion: v1
kind: Service
metadata:
  name: example
spec:
  selector:
    app: example
  ports:
    - name: http
      port: 8080
      targetPort: 5678
```

Apply the application:

```bash
kubectl apply -f example-app.yaml
```

The Ingress backend may reference the Service port by number or name. The adapter resolves it to `Service.spec.ports[].port`, not `targetPort`.

## Step 4: Create an IngressClass

The IngressClass is cluster-scoped and normally only needs to be created once per cluster.

```yaml
apiVersion: networking.k8s.io/v1
kind: IngressClass
metadata:
  name: frp
spec:
  controller: frp.aureum.cloud/ingress-controller
```

Apply it:

```bash
kubectl apply -f ingressclass.yaml
```

## Step 5: Create an Ingress

The `frp.aureum.cloud/exit-server` annotation must refer to an ExitServer in the same namespace as the Ingress.

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: example
  annotations:
    frp.aureum.cloud/exit-server: exit-server-sample
spec:
  ingressClassName: frp
  rules:
    - host: example.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: example
                port:
                  number: 8080
```

Apply it:

```bash
kubectl apply -f ingress.yaml
```

## Step 6: Verify the generated Tunnel

List Tunnels generated for the Ingress:

```bash
kubectl get tunnels -l frp.aureum.cloud/ingress-name=example
```

Inspect the generated Tunnel:

```bash
kubectl get tunnel <generated-name> -o yaml
```

Its specification should look similar to this:

```yaml
spec:
  exitServer: exit-server-sample
  http:
    customDomains:
      - example.com
    locations:
      - /
    serviceRef:
      name: example
      namespace: default
    localPort: 8080
```

You can also verify that the Tunnel is owned by the Ingress:

```bash
kubectl get tunnel <generated-name> \
  -o jsonpath='{.metadata.ownerReferences}'
```

After frps applies the configuration, traffic for `example.com` will be forwarded to the `example` Service. The public URL may require the frps HTTP virtual-host port, depending on your frps configuration.

## Named Service port example

Instead of a numeric port, an Ingress can reference a named Service port:

```yaml
backend:
  service:
    name: example
    port:
      name: http
```

The adapter reads the Service and resolves `http` to port `8080`.

## Multiple paths and Services

An Ingress can route paths to different Services. The adapter creates an independent Tunnel for each backend:

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: multi-path
  annotations:
    frp.aureum.cloud/exit-server: exit-server-sample
spec:
  ingressClassName: frp
  rules:
    - host: app.example.com
      http:
        paths:
          - path: /api
            pathType: Prefix
            backend:
              service:
                name: api-service
                port:
                  name: http
          - path: /
            pathType: Prefix
            backend:
              service:
                name: web-service
                port:
                  number: 80
```

## Update and deletion behavior

- **Update an Ingress:** Changes to its host, path, backend Service, or port update the generated Tunnels. Obsolete Tunnels are deleted.
- **Delete an Ingress:** Kubernetes garbage collection removes the generated Tunnels through their owner references.
- **Change to another IngressClass:** The adapter stops managing the Ingress and deletes the Tunnels that it previously generated.

## Current limitations

The first version supports HTTP rules with non-empty hosts and the `Prefix` or `ImplementationSpecific` path types.

The following configurations are rejected:

| Configuration | Reason |
| --- | --- |
| `spec.tls` | The current HTTP Tunnel model does not represent Ingress TLS termination semantics. |
| `spec.defaultBackend` | A default backend does not provide the domain required by an FRP HTTP Tunnel. |
| A rule without `host` | The Tunnel requires `customDomains` or `subdomain`. |
| `pathType: Exact` | FRP `locations` cannot guarantee Kubernetes Exact matching semantics. |
| A resource backend instead of a Service | The Tunnel model only supports Kubernetes Service backends. |

For HTTPS or TLS termination, use a single Tunnel pointing to an existing Ingress controller Service, such as Traefik or NGINX, and let that controller handle TLS and advanced routing. See `config/samples/frp_traefik_http_plugin.yaml` for an example.

## Troubleshooting

### No Tunnel is created

Check the IngressClass:

```bash
kubectl get ingressclass frp -o yaml
```

It must contain:

```yaml
spec:
  controller: frp.aureum.cloud/ingress-controller
```

Also verify that the Ingress uses it:

```yaml
spec:
  ingressClassName: frp
```

### ExitServer not found

Verify the annotation and namespace:

```yaml
metadata:
  annotations:
    frp.aureum.cloud/exit-server: exit-server-sample
```

The Ingress and ExitServer must be in the same namespace.

### Service port cannot be resolved

Confirm that the referenced port number or name exists in `Service.spec.ports`:

```bash
kubectl get service example -o yaml
```

### Tunnel exists but traffic does not reach the Service

Check the frpc Pod and controller logs:

```bash
kubectl get pods -l app.kubernetes.io/created-by=exit-server-sample
kubectl logs -l app.kubernetes.io/created-by=exit-server-sample
```

Also verify that frps has HTTP virtual-host routing configured, including `vhostHTTPPort` where required.
