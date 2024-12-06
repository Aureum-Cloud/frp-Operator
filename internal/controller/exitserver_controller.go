/*
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
*/

package controller

import (
	"context"
	"fmt"
	"frp-operator/internal/clients"
	"frp-operator/internal/constants"
	"frp-operator/internal/templates"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	frpv1 "frp-operator/api/v1"
)

// ExitServerReconciler reconciles a ExitServer object
type ExitServerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=frp.aureum.cloud,resources=exitservers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=frp.aureum.cloud,resources=exitservers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=frp.aureum.cloud,resources=exitservers/finalizers,verbs=update

//+kubebuilder:rbac:groups=frp.aureum.cloud,resources=tunnels,verbs=get;list;watch

//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.15.0/pkg/reconcile
func (r *ExitServerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	exitServer := &frpv1.ExitServer{}
	err := r.Client.Get(ctx, req.NamespacedName, exitServer)
	if err != nil {
		logger.Error(err, "Failed to fetch FRP ExitServer resource definition")
		return ctrl.Result{}, err
	}

	token, err := r.getTokenFromSecret(ctx, exitServer)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Secret with token not found")
		return ctrl.Result{RequeueAfter: constants.RequeueAfterSeconds * time.Second}, nil
	} else if err != nil {
		logger.Error(err, "Failed to fetch secret")
		return ctrl.Result{}, err
	}

	tunnels := &frpv1.TunnelList{}
	err = r.Client.List(ctx, tunnels)
	if err != nil {
		logger.Error(err, "Failed to fetch FRP Tunnel resource definitions")
		return ctrl.Result{}, err
	}

	var relatedTunnels []frpv1.Tunnel
	for _, tunnel := range tunnels.Items {
		if tunnel.Spec.ExitServer == exitServer.Name {
			relatedTunnels = append(relatedTunnels, tunnel)
		}
	}

	configuration, err := templates.CreateConfiguration(exitServer, token, relatedTunnels)
	if err != nil {
		logger.Error(err, "Failed to create FRP configuration")
		return ctrl.Result{}, err
	}

	secretResult, err := r.createSecret(ctx, exitServer, configuration)
	if err != nil {
		logger.Error(err, "Failed to create or update FRP secret")
		return ctrl.Result{}, err
	}
	if secretResult != controllerutil.OperationResultNone {
		logger.Info("Secret: " + string(secretResult))
	}

	serviceResult, svc, err := r.createService(ctx, exitServer)
	if err != nil {
		logger.Error(err, "Failed to create or update FRP service")
		return ctrl.Result{}, err
	}
	if serviceResult != controllerutil.OperationResultNone {
		logger.Info("Service: " + string(serviceResult))
	}

	podResult, pod, err := r.createPod(ctx, exitServer)
	if err != nil {
		logger.Error(err, "Failed to create or update FRP pod")
		return ctrl.Result{}, err
	}
	if podResult != controllerutil.OperationResultNone {
		logger.Info("Pod: " + string(podResult))
	}

	if pod.Status.Phase != v1.PodRunning {
		logger.Info("Pod is not running, requeuing reconciliation")
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}

	if secretResult == controllerutil.OperationResultUpdated {
		waitingSeconds := 0
		configURL := fmt.Sprintf("http://%s.%s.svc:%d/api/config", svc.Name, svc.Namespace, constants.AdminAPIPort)
		reloadURL := fmt.Sprintf("http://%s.%s.svc:%d/api/reload", svc.Name, svc.Namespace, constants.AdminAPIPort)

		logger.Info("Waiting on configuration update...")

		for {
			time.Sleep(time.Second)
			waitingSeconds++

			appliedConfiguration, err := clients.ExecuteHttpRequest("GET", configURL)
			if err != nil {
				logger.Error(err, "Failed to get applied FRP configuration")
				return ctrl.Result{}, err
			}

			if appliedConfiguration == configuration {
				break
			}

			if waitingSeconds >= constants.CompareConfigTimeoutSeconds {
				return ctrl.Result{}, fmt.Errorf("configuration not updated after waiting '%d' seconds", waitingSeconds)
			}
		}

		logger.Info("Updated FRP configuration successfully")

		if _, err := clients.ExecuteHttpRequest("GET", reloadURL); err != nil {
			logger.Error(err, "Failed to reload FRP configuration")
			return ctrl.Result{}, err
		}

		logger.Info("Reloaded FRP configuration successfully")
	}

	return ctrl.Result{RequeueAfter: constants.RequeueAfterSeconds * time.Second}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ExitServerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&frpv1.ExitServer{}).
		Complete(r)
}

func (r *ExitServerReconciler) getTokenFromSecret(ctx context.Context, exitServer *frpv1.ExitServer) (string, error) {
	name := exitServer.Spec.Authentication.Token.Secret.Name
	key := exitServer.Spec.Authentication.Token.Secret.Key

	secret := &v1.Secret{}
	err := r.Client.Get(ctx, types.NamespacedName{Name: name, Namespace: exitServer.Namespace}, secret)
	if err != nil {
		return "", err
	}

	tokenBytes, ok := secret.Data[key]
	if !ok {
		return "", fmt.Errorf("token key '%s' not found in secret data", key)
	}

	return string(tokenBytes), nil
}

func (r *ExitServerReconciler) createSecret(ctx context.Context, exitServer *frpv1.ExitServer, configuration string) (controllerutil.OperationResult, error) {
	configurationFilename := "config.toml"

	secret := &v1.Secret{
		ObjectMeta: getObjectMeta(exitServer),
		StringData: map[string]string{
			configurationFilename: configuration,
		},
	}

	if err := controllerutil.SetControllerReference(exitServer, secret, r.Scheme); err != nil {
		return controllerutil.OperationResultNone, err
	}

	return controllerutil.CreateOrUpdate(ctx, r.Client, secret, func() error {
		if string(secret.Data[configurationFilename]) != configuration {
			secret.StringData = map[string]string{
				configurationFilename: configuration,
			}
		}
		return nil
	})
}

func (r *ExitServerReconciler) createPod(ctx context.Context, exitServer *frpv1.ExitServer) (controllerutil.OperationResult, *v1.Pod, error) {
	po := &v1.Pod{
		ObjectMeta: getObjectMeta(exitServer),
		Spec: v1.PodSpec{
			Containers: []v1.Container{{
				Name:    "frpc",
				Image:   constants.ClientImage,
				Command: []string{"frpc", "-c", "/frp/config.toml"},
				Ports: []v1.ContainerPort{{
					ContainerPort: constants.AdminAPIPort,
				}},
				VolumeMounts: []v1.VolumeMount{{
					Name:      exitServer.Name + "-frpc",
					MountPath: "/frp",
				}},
			}},
			Volumes: []v1.Volume{{
				Name: exitServer.Name + "-frpc",
				VolumeSource: v1.VolumeSource{
					Secret: &v1.SecretVolumeSource{
						SecretName: exitServer.Name + "-frpc",
					},
				},
			}},
		},
	}

	if err := controllerutil.SetControllerReference(exitServer, po, r.Scheme); err != nil {
		return controllerutil.OperationResultNone, po, err
	}

	update, err := controllerutil.CreateOrUpdate(ctx, r.Client, po, func() error {
		return nil
	})

	return update, po, err
}

func (r *ExitServerReconciler) createService(ctx context.Context, exitServer *frpv1.ExitServer) (controllerutil.OperationResult, *v1.Service, error) {
	svc := &v1.Service{
		ObjectMeta: getObjectMeta(exitServer),
		Spec: v1.ServiceSpec{
			Type:     "ClusterIP",
			Selector: getLabels(exitServer),
			Ports: []v1.ServicePort{{
				Name:       "admin-api",
				Port:       constants.AdminAPIPort,
				TargetPort: intstr.FromInt32(constants.AdminAPIPort),
				Protocol:   v1.ProtocolTCP,
			}},
		},
	}

	if err := controllerutil.SetControllerReference(exitServer, svc, r.Scheme); err != nil {
		return controllerutil.OperationResultNone, svc, err
	}

	update, err := controllerutil.CreateOrUpdate(ctx, r.Client, svc, func() error {
		return nil
	})

	return update, svc, err
}

func getObjectMeta(exitServer *frpv1.ExitServer) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:      exitServer.Name + "-frpc",
		Namespace: exitServer.Namespace,
		Labels:    getLabels(exitServer),
	}
}

func getLabels(exitServer *frpv1.ExitServer) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":       exitServer.Name + "-frpc",
		"app.kubernetes.io/component":  "frp-client",
		"app.kubernetes.io/managed-by": "frp-operator",
		"app.kubernetes.io/created-by": exitServer.Name,
	}
}
