/*
Copyright 2026 Aureum Cloud, N-Bit, Niek Berenschot.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	frpv1 "frp-operator/api/v1"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	IngressControllerName        = "frp.aureum.cloud/ingress-controller"
	ExitServerAnnotation         = "frp.aureum.cloud/exit-server"
	ingressNameLabel             = "frp.aureum.cloud/ingress-name"
	ingressAdapterManagedByLabel = "app.kubernetes.io/managed-by"
	ingressAdapterManagedByValue = "frp-ingress-adapter"
)

// IngressReconciler translates Kubernetes Ingress rules into FRP HTTP Tunnels.
type IngressReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses;ingressclasses,verbs=get;list;watch
// +kubebuilder:rbac:groups=frp.aureum.cloud,resources=tunnels,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=frp.aureum.cloud,resources=exitservers,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch

func (r *IngressReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	ingress := &networkingv1.Ingress{}
	if err := r.Get(ctx, req.NamespacedName, ingress); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	managed, err := r.isManaged(ctx, ingress)
	if err != nil {
		return ctrl.Result{}, err
	}
	if !managed {
		return ctrl.Result{}, r.deleteStaleTunnels(ctx, ingress, nil)
	}

	desired, err := r.desiredTunnels(ctx, ingress)
	if err != nil {
		return ctrl.Result{}, err
	}

	desiredNames := make(map[string]struct{}, len(desired))
	for i := range desired {
		tunnel := desired[i]
		desiredNames[tunnel.Name] = struct{}{}
		if err := r.createOrUpdateTunnel(ctx, ingress, tunnel); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, r.deleteStaleTunnels(ctx, ingress, desiredNames)
}

func (r *IngressReconciler) isManaged(ctx context.Context, ingress *networkingv1.Ingress) (bool, error) {
	if ingress.Spec.IngressClassName == nil || *ingress.Spec.IngressClassName == "" {
		return false, nil
	}

	ingressClass := &networkingv1.IngressClass{}
	if err := r.Get(ctx, types.NamespacedName{Name: *ingress.Spec.IngressClassName}, ingressClass); err != nil {
		if apierrors.IsNotFound(err) {
			return false, nil
		}
		return false, fmt.Errorf("get IngressClass %q: %w", *ingress.Spec.IngressClassName, err)
	}
	return ingressClass.Spec.Controller == IngressControllerName, nil
}

func (r *IngressReconciler) desiredTunnels(ctx context.Context, ingress *networkingv1.Ingress) ([]*frpv1.Tunnel, error) {
	exitServerName := ingress.Annotations[ExitServerAnnotation]
	if exitServerName == "" {
		return nil, fmt.Errorf("Ingress %s/%s is missing annotation %q", ingress.Namespace, ingress.Name, ExitServerAnnotation)
	}
	if len(ingress.Spec.TLS) > 0 {
		return nil, fmt.Errorf("Ingress %s/%s uses TLS, which is not supported by the FRP Ingress adapter", ingress.Namespace, ingress.Name)
	}
	if ingress.Spec.DefaultBackend != nil {
		return nil, fmt.Errorf("Ingress %s/%s uses defaultBackend, which is not supported by the FRP Ingress adapter", ingress.Namespace, ingress.Name)
	}

	exitServer := &frpv1.ExitServer{}
	if err := r.Get(ctx, types.NamespacedName{Namespace: ingress.Namespace, Name: exitServerName}, exitServer); err != nil {
		return nil, fmt.Errorf("get ExitServer %s/%s: %w", ingress.Namespace, exitServerName, err)
	}

	var tunnels []*frpv1.Tunnel
	for _, rule := range ingress.Spec.Rules {
		if rule.Host == "" {
			return nil, fmt.Errorf("Ingress %s/%s contains a rule without a host", ingress.Namespace, ingress.Name)
		}
		if rule.HTTP == nil {
			return nil, fmt.Errorf("Ingress %s/%s contains a non-HTTP rule for host %q", ingress.Namespace, ingress.Name, rule.Host)
		}
		for _, path := range rule.HTTP.Paths {
			if path.Backend.Service == nil {
				return nil, fmt.Errorf("Ingress %s/%s path %q uses an unsupported resource backend", ingress.Namespace, ingress.Name, path.Path)
			}
			if path.PathType != nil && *path.PathType == networkingv1.PathTypeExact {
				return nil, fmt.Errorf("Ingress %s/%s path %q uses Exact pathType, which FRP locations cannot preserve", ingress.Namespace, ingress.Name, path.Path)
			}

			port, err := r.resolveServicePort(ctx, ingress.Namespace, path.Backend.Service)
			if err != nil {
				return nil, fmt.Errorf("resolve backend for host %q path %q: %w", rule.Host, path.Path, err)
			}
			pathValue := path.Path
			if pathValue == "" {
				pathValue = "/"
			}
			namespace := ingress.Namespace
			tunnels = append(tunnels, &frpv1.Tunnel{
				ObjectMeta: metav1.ObjectMeta{
					Name:      tunnelName(ingress.Name, rule.Host, pathValue, path.Backend.Service.Name, port),
					Namespace: ingress.Namespace,
					Labels: map[string]string{
						ingressNameLabel:             ingress.Name,
						ingressAdapterManagedByLabel: ingressAdapterManagedByValue,
					},
				},
				Spec: frpv1.TunnelSpec{
					ExitServer: exitServer.Name,
					HTTP: &frpv1.HTTP{
						CustomDomains: []string{rule.Host},
						Locations:     []string{pathValue},
						ServiceRef: &frpv1.ServiceRef{
							Name:      path.Backend.Service.Name,
							Namespace: &namespace,
						},
						LocalPort: &port,
					},
				},
			})
		}
	}
	return tunnels, nil
}

func (r *IngressReconciler) resolveServicePort(ctx context.Context, namespace string, backend *networkingv1.IngressServiceBackend) (int, error) {
	service := &corev1.Service{}
	if err := r.Get(ctx, types.NamespacedName{Namespace: namespace, Name: backend.Name}, service); err != nil {
		return 0, fmt.Errorf("get Service %s/%s: %w", namespace, backend.Name, err)
	}

	for _, port := range service.Spec.Ports {
		if backend.Port.Name != "" && port.Name == backend.Port.Name {
			return int(port.Port), nil
		}
		if backend.Port.Number != 0 && port.Port == backend.Port.Number {
			return int(port.Port), nil
		}
	}
	return 0, fmt.Errorf("Service %s/%s does not expose backend port %v", namespace, backend.Name, backend.Port)
}

func (r *IngressReconciler) createOrUpdateTunnel(ctx context.Context, ingress *networkingv1.Ingress, desired *frpv1.Tunnel) error {
	current := &frpv1.Tunnel{ObjectMeta: metav1.ObjectMeta{Name: desired.Name, Namespace: desired.Namespace}}
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, current, func() error {
		current.Labels = desired.Labels
		current.Spec = desired.Spec
		return controllerutil.SetControllerReference(ingress, current, r.Scheme)
	})
	return err
}

func (r *IngressReconciler) deleteStaleTunnels(ctx context.Context, ingress *networkingv1.Ingress, desiredNames map[string]struct{}) error {
	list := &frpv1.TunnelList{}
	if err := r.List(ctx, list, client.InNamespace(ingress.Namespace), client.MatchingLabels{
		ingressNameLabel:             ingress.Name,
		ingressAdapterManagedByLabel: ingressAdapterManagedByValue,
	}); err != nil {
		return err
	}

	for i := range list.Items {
		tunnel := &list.Items[i]
		if !metav1.IsControlledBy(tunnel, ingress) {
			continue
		}
		if _, ok := desiredNames[tunnel.Name]; ok {
			continue
		}
		if err := r.Delete(ctx, tunnel); err != nil && !apierrors.IsNotFound(err) {
			return err
		}
	}
	return nil
}

func tunnelName(ingressName, host, path, serviceName string, port int) string {
	identity := strings.Join([]string{host, path, serviceName, fmt.Sprint(port)}, "\x00")
	sum := sha256.Sum256([]byte(identity))
	suffix := hex.EncodeToString(sum[:])[:10]
	prefix := strings.Trim(strings.ToLower(ingressName), "-")
	if len(prefix) > 52 {
		prefix = strings.TrimRight(prefix[:52], "-")
	}
	return prefix + "-" + suffix
}

func (r *IngressReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&networkingv1.Ingress{}).
		Owns(&frpv1.Tunnel{}).
		Named("ingress-adapter").
		Complete(r)
}
