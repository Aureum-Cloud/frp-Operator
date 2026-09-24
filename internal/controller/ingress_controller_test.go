package controller

import (
	"context"
	"sort"
	"strings"
	"testing"

	frpv1 "frp-operator/api/v1"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	testNamespace  = "apps"
	testIngress    = "web"
	testClass      = "frp"
	testExitServer = "public"
)

func TestIngressReconcilerCreatesTunnel(t *testing.T) {
	ingress := newIngress(rule("web.example.com", path("/api", "web", namedPort("http"))))
	c := newTestClient(t, managedClass(), exitServer(), service("web", servicePort("http", 8080)), ingress)
	r := newReconciler(t, c)

	reconcile(t, r)

	tunnel := singleTunnel(t, c)
	if tunnel.Spec.ExitServer != testExitServer || tunnel.Spec.HTTP == nil {
		t.Fatalf("unexpected tunnel spec: %#v", tunnel.Spec)
	}
	if got := *tunnel.Spec.HTTP.LocalPort; got != 8080 {
		t.Errorf("local port = %d, want 8080", got)
	}
	if got := tunnel.Spec.HTTP.Locations[0]; got != "/api" {
		t.Errorf("location = %q, want /api", got)
	}
	if got := tunnel.Spec.HTTP.CustomDomains[0]; got != "web.example.com" {
		t.Errorf("custom domain = %q, want web.example.com", got)
	}
	if !metav1.IsControlledBy(tunnel, ingress) {
		t.Error("tunnel is not controlled by ingress")
	}
}

func TestIngressReconcilerIgnoresOtherClass(t *testing.T) {
	ingress := newIngress()
	ingress.Spec.IngressClassName = ptr("other")
	c := newTestClient(t, ingress, unmanagedClass("other"))
	r := newReconciler(t, c)

	reconcile(t, r)

	if names := tunnelNames(t, c); len(names) != 0 {
		t.Fatalf("tunnels = %v, want none", names)
	}
}

func TestIngressReconcilerRejectsTLS(t *testing.T) {
	ingress := newIngress(rule("web.example.com", path("/", "web", numberPort(8080))))
	ingress.Spec.TLS = []networkingv1.IngressTLS{{Hosts: []string{"web.example.com"}}}
	c := newTestClient(t, managedClass(), exitServer(), service("web", servicePort("http", 8080)), ingress)
	r := newReconciler(t, c)

	if _, err := r.Reconcile(context.Background(), request()); err == nil {
		t.Fatal("Reconcile() error = nil, want TLS unsupported error")
	}
}

// --- Update behavior -------------------------------------------------------

func TestIngressReconcilerIsIdempotent(t *testing.T) {
	c := newTestClient(t, managedClass(), exitServer(), service("web", servicePort("http", 8080)),
		newIngress(rule("web.example.com", path("/api", "web", namedPort("http")))))
	r := newReconciler(t, c)

	reconcile(t, r)
	first := singleTunnel(t, c)

	reconcile(t, r)
	second := singleTunnel(t, c)

	if first.Name != second.Name {
		t.Fatalf("tunnel renamed from %q to %q", first.Name, second.Name)
	}
	if first.ResourceVersion != second.ResourceVersion {
		t.Errorf("tunnel was rewritten: resourceVersion %q -> %q", first.ResourceVersion, second.ResourceVersion)
	}
	if !strings.HasPrefix(second.Name, testIngress+"-") {
		t.Errorf("tunnel name %q is not derived from ingress name", second.Name)
	}
}

func TestIngressReconcilerUpdatesTunnelsOnSpecChange(t *testing.T) {
	tests := []struct {
		name         string
		updated      networkingv1.IngressRule
		wantHost     string
		wantLocation string
		wantService  string
		wantPort     int
	}{
		{
			name:         "host changed",
			updated:      rule("new.example.com", path("/api", "web", namedPort("http"))),
			wantHost:     "new.example.com",
			wantLocation: "/api",
			wantService:  "web",
			wantPort:     8080,
		},
		{
			name:         "path changed",
			updated:      rule("web.example.com", path("/v2", "web", namedPort("http"))),
			wantHost:     "web.example.com",
			wantLocation: "/v2",
			wantService:  "web",
			wantPort:     8080,
		},
		{
			name:         "backend service changed",
			updated:      rule("web.example.com", path("/api", "api", namedPort("http"))),
			wantHost:     "web.example.com",
			wantLocation: "/api",
			wantService:  "api",
			wantPort:     9090,
		},
		{
			name:         "backend port changed",
			updated:      rule("web.example.com", path("/api", "web", numberPort(8081))),
			wantHost:     "web.example.com",
			wantLocation: "/api",
			wantService:  "web",
			wantPort:     8081,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := newTestClient(t, managedClass(), exitServer(),
				service("web", servicePort("http", 8080), servicePort("alt", 8081)),
				service("api", servicePort("http", 9090)),
				newIngress(rule("web.example.com", path("/api", "web", namedPort("http")))))
			r := newReconciler(t, c)

			reconcile(t, r)
			original := singleTunnel(t, c)

			updateIngressRules(t, c, test.updated)
			reconcile(t, r)

			tunnel := singleTunnel(t, c)
			if tunnel.Name == original.Name {
				t.Fatalf("expected a new tunnel identity, still %q", tunnel.Name)
			}
			if got := tunnel.Spec.HTTP.CustomDomains[0]; got != test.wantHost {
				t.Errorf("host = %q, want %q", got, test.wantHost)
			}
			if got := tunnel.Spec.HTTP.Locations[0]; got != test.wantLocation {
				t.Errorf("location = %q, want %q", got, test.wantLocation)
			}
			if got := tunnel.Spec.HTTP.ServiceRef.Name; got != test.wantService {
				t.Errorf("service = %q, want %q", got, test.wantService)
			}
			if got := *tunnel.Spec.HTTP.LocalPort; got != test.wantPort {
				t.Errorf("port = %d, want %d", got, test.wantPort)
			}
		})
	}
}

func TestIngressReconcilerAddsAndRemovesPaths(t *testing.T) {
	c := newTestClient(t, managedClass(), exitServer(),
		service("web", servicePort("http", 8080)),
		service("api", servicePort("http", 9090)),
		newIngress(rule("web.example.com", path("/", "web", namedPort("http")))))
	r := newReconciler(t, c)

	reconcile(t, r)
	if got := len(tunnelNames(t, c)); got != 1 {
		t.Fatalf("tunnels = %d, want 1", got)
	}

	updateIngressRules(t, c, rule("web.example.com",
		path("/", "web", namedPort("http")),
		path("/api", "api", namedPort("http")),
	))
	reconcile(t, r)
	if got := len(tunnelNames(t, c)); got != 2 {
		t.Fatalf("tunnels after add = %d, want 2", got)
	}

	updateIngressRules(t, c, rule("web.example.com", path("/api", "api", namedPort("http"))))
	reconcile(t, r)

	tunnel := singleTunnel(t, c)
	if got := tunnel.Spec.HTTP.Locations[0]; got != "/api" {
		t.Errorf("remaining location = %q, want /api", got)
	}
	if got := tunnel.Spec.HTTP.ServiceRef.Name; got != "api" {
		t.Errorf("remaining service = %q, want api", got)
	}
}

func TestIngressReconcilerRemovesAllTunnelsWhenRulesCleared(t *testing.T) {
	c := newTestClient(t, managedClass(), exitServer(), service("web", servicePort("http", 8080)),
		newIngress(rule("web.example.com", path("/", "web", namedPort("http")))))
	r := newReconciler(t, c)

	reconcile(t, r)
	updateIngressRules(t, c)
	reconcile(t, r)

	if names := tunnelNames(t, c); len(names) != 0 {
		t.Fatalf("tunnels = %v, want none", names)
	}
}

// --- Deletion behavior -----------------------------------------------------

func TestIngressReconcilerSetsOwnerReferenceForGarbageCollection(t *testing.T) {
	ingress := newIngress(rule("web.example.com", path("/", "web", namedPort("http"))))
	c := newTestClient(t, managedClass(), exitServer(), service("web", servicePort("http", 8080)), ingress)
	r := newReconciler(t, c)

	reconcile(t, r)

	tunnel := singleTunnel(t, c)
	refs := tunnel.GetOwnerReferences()
	if len(refs) != 1 {
		t.Fatalf("owner references = %d, want 1", len(refs))
	}
	ref := refs[0]
	if ref.Kind != "Ingress" || ref.Name != testIngress || ref.UID != ingress.UID {
		t.Errorf("unexpected owner reference: %#v", ref)
	}
	if ref.Controller == nil || !*ref.Controller {
		t.Error("owner reference is not a controller reference")
	}
	if ref.BlockOwnerDeletion == nil || !*ref.BlockOwnerDeletion {
		t.Error("owner reference does not block owner deletion")
	}
	if tunnel.Namespace != ingress.Namespace {
		t.Errorf("tunnel namespace = %q, want %q", tunnel.Namespace, ingress.Namespace)
	}
}

func TestIngressReconcilerIgnoresMissingIngress(t *testing.T) {
	c := newTestClient(t, managedClass(), exitServer())
	r := newReconciler(t, c)

	if _, err := r.Reconcile(context.Background(), request()); err != nil {
		t.Fatalf("Reconcile() error = %v, want nil for deleted ingress", err)
	}
}

func TestIngressReconcilerLeavesOrphanTunnelsToGarbageCollection(t *testing.T) {
	orphan := &frpv1.Tunnel{ObjectMeta: metav1.ObjectMeta{
		Name:      "web-orphan",
		Namespace: testNamespace,
		Labels: map[string]string{
			ingressNameLabel:             testIngress,
			ingressAdapterManagedByLabel: ingressAdapterManagedByValue,
		},
		OwnerReferences: []metav1.OwnerReference{{
			APIVersion: "networking.k8s.io/v1",
			Kind:       "Ingress",
			Name:       testIngress,
			UID:        types.UID("deleted-ingress"),
			Controller: ptr(true),
		}},
	}}
	c := newTestClient(t, orphan)
	r := newReconciler(t, c)

	reconcile(t, r)

	if names := tunnelNames(t, c); len(names) != 1 {
		t.Fatalf("tunnels = %v, want the orphan to be left for garbage collection", names)
	}
}

// --- IngressClass change behavior ------------------------------------------

func TestIngressReconcilerDeletesTunnelsWhenNoLongerManaged(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*networkingv1.Ingress)
		extra  []client.Object
	}{
		{
			name:   "class switched to another controller",
			mutate: func(i *networkingv1.Ingress) { i.Spec.IngressClassName = ptr("other") },
			extra:  []client.Object{unmanagedClass("other")},
		},
		{
			name:   "class name removed",
			mutate: func(i *networkingv1.Ingress) { i.Spec.IngressClassName = nil },
		},
		{
			name:   "class name emptied",
			mutate: func(i *networkingv1.Ingress) { i.Spec.IngressClassName = ptr("") },
		},
		{
			name:   "class deleted from cluster",
			mutate: func(i *networkingv1.Ingress) { i.Spec.IngressClassName = ptr("missing") },
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			objects := append([]client.Object{
				managedClass(), exitServer(), service("web", servicePort("http", 8080)),
				newIngress(rule("web.example.com", path("/", "web", namedPort("http")))),
			}, test.extra...)
			c := newTestClient(t, objects...)
			r := newReconciler(t, c)

			reconcile(t, r)
			if got := len(tunnelNames(t, c)); got != 1 {
				t.Fatalf("tunnels before class change = %d, want 1", got)
			}

			ingress := getIngress(t, c)
			test.mutate(ingress)
			if err := c.Update(context.Background(), ingress); err != nil {
				t.Fatal(err)
			}

			reconcile(t, r)

			if names := tunnelNames(t, c); len(names) != 0 {
				t.Fatalf("tunnels = %v, want all removed after the ingress became unmanaged", names)
			}
		})
	}
}

func TestIngressReconcilerCleanupIsScopedToItsOwnTunnels(t *testing.T) {
	foreign := &frpv1.Tunnel{ObjectMeta: metav1.ObjectMeta{
		Name:      "foreign-adapter-tunnel",
		Namespace: testNamespace,
		Labels: map[string]string{
			ingressNameLabel:             testIngress,
			ingressAdapterManagedByLabel: ingressAdapterManagedByValue,
		},
		OwnerReferences: []metav1.OwnerReference{{
			APIVersion: "networking.k8s.io/v1",
			Kind:       "Ingress",
			Name:       "another-ingress",
			UID:        types.UID("another-uid"),
			Controller: ptr(true),
		}},
	}}
	manual := &frpv1.Tunnel{ObjectMeta: metav1.ObjectMeta{
		Name:      "hand-written-tunnel",
		Namespace: testNamespace,
	}}

	ingress := newIngress()
	ingress.Spec.IngressClassName = ptr("other")
	c := newTestClient(t, ingress, unmanagedClass("other"), foreign, manual)
	r := newReconciler(t, c)

	reconcile(t, r)

	names := tunnelNames(t, c)
	if len(names) != 2 {
		t.Fatalf("tunnels = %v, want foreign and manual tunnels preserved", names)
	}
}

// --- helpers ---------------------------------------------------------------

func newReconciler(t *testing.T, c client.Client) *IngressReconciler {
	t.Helper()
	return &IngressReconciler{Client: c, Scheme: testScheme(t)}
}

func newTestClient(t *testing.T, objects ...client.Object) client.Client {
	t.Helper()
	return fake.NewClientBuilder().WithScheme(testScheme(t)).WithObjects(objects...).Build()
}

func request() ctrl.Request {
	return ctrl.Request{NamespacedName: types.NamespacedName{Name: testIngress, Namespace: testNamespace}}
}

func reconcile(t *testing.T, r *IngressReconciler) {
	t.Helper()
	if _, err := r.Reconcile(context.Background(), request()); err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}
}

func newIngress(rules ...networkingv1.IngressRule) *networkingv1.Ingress {
	return &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:        testIngress,
			Namespace:   testNamespace,
			UID:         types.UID("ingress-uid"),
			Annotations: map[string]string{ExitServerAnnotation: testExitServer},
		},
		Spec: networkingv1.IngressSpec{
			IngressClassName: ptr(testClass),
			Rules:            rules,
		},
	}
}

func rule(host string, paths ...networkingv1.HTTPIngressPath) networkingv1.IngressRule {
	return networkingv1.IngressRule{
		Host: host,
		IngressRuleValue: networkingv1.IngressRuleValue{
			HTTP: &networkingv1.HTTPIngressRuleValue{Paths: paths},
		},
	}
}

func path(urlPath, serviceName string, port networkingv1.ServiceBackendPort) networkingv1.HTTPIngressPath {
	pathType := networkingv1.PathTypePrefix
	return networkingv1.HTTPIngressPath{
		Path:     urlPath,
		PathType: &pathType,
		Backend: networkingv1.IngressBackend{
			Service: &networkingv1.IngressServiceBackend{Name: serviceName, Port: port},
		},
	}
}

func namedPort(name string) networkingv1.ServiceBackendPort {
	return networkingv1.ServiceBackendPort{Name: name}
}

func numberPort(number int32) networkingv1.ServiceBackendPort {
	return networkingv1.ServiceBackendPort{Number: number}
}

func service(name string, ports ...corev1.ServicePort) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: testNamespace},
		Spec:       corev1.ServiceSpec{Ports: ports},
	}
}

func servicePort(name string, port int32) corev1.ServicePort {
	return corev1.ServicePort{Name: name, Port: port}
}

func exitServer() *frpv1.ExitServer {
	return &frpv1.ExitServer{ObjectMeta: metav1.ObjectMeta{Name: testExitServer, Namespace: testNamespace}}
}

func managedClass() *networkingv1.IngressClass {
	return &networkingv1.IngressClass{
		ObjectMeta: metav1.ObjectMeta{Name: testClass},
		Spec:       networkingv1.IngressClassSpec{Controller: IngressControllerName},
	}
}

func unmanagedClass(name string) *networkingv1.IngressClass {
	return &networkingv1.IngressClass{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec:       networkingv1.IngressClassSpec{Controller: "example.com/other"},
	}
}

func getIngress(t *testing.T, c client.Client) *networkingv1.Ingress {
	t.Helper()
	ingress := &networkingv1.Ingress{}
	if err := c.Get(context.Background(), request().NamespacedName, ingress); err != nil {
		t.Fatal(err)
	}
	return ingress
}

func updateIngressRules(t *testing.T, c client.Client, rules ...networkingv1.IngressRule) {
	t.Helper()
	ingress := getIngress(t, c)
	ingress.Spec.Rules = rules
	if err := c.Update(context.Background(), ingress); err != nil {
		t.Fatal(err)
	}
}

func tunnelNames(t *testing.T, c client.Client) []string {
	t.Helper()
	list := &frpv1.TunnelList{}
	if err := c.List(context.Background(), list); err != nil {
		t.Fatal(err)
	}
	names := make([]string, 0, len(list.Items))
	for _, tunnel := range list.Items {
		names = append(names, tunnel.Name)
	}
	sort.Strings(names)
	return names
}

func singleTunnel(t *testing.T, c client.Client) *frpv1.Tunnel {
	t.Helper()
	list := &frpv1.TunnelList{}
	if err := c.List(context.Background(), list); err != nil {
		t.Fatal(err)
	}
	if len(list.Items) != 1 {
		t.Fatalf("tunnels = %d, want 1", len(list.Items))
	}
	return &list.Items[0]
}

func ptr[T any](value T) *T {
	return &value
}

func testScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	if err := networkingv1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	if err := frpv1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	return scheme
}
