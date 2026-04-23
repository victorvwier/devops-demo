package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	demov1alpha1 "devops-demo/operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	frontendName       = "tiny-llm-frontend"
	frontendConfigMap  = "tiny-llm-frontend-config"
	frontendImage      = "ghcr.io/victorvwier/tiny-llm-runner:latest"
	frontendHost       = "tiny-llm.demo.example.com"
	backendImage       = "ghcr.io/ggml-org/llama.cpp:server"
	backendListenPort  = 8080
	frontendListenPort = 8080
)

type TinyLLMServiceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *TinyLLMServiceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithValues("namespace", req.Namespace, "name", req.Name)
	logger.Info("reconciling tiny llm service")

	var svc demov1alpha1.TinyLLMService
	if err := r.Get(ctx, req.NamespacedName, &svc); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	replicas := int32(1)
	if svc.Spec.Replicas != nil {
		replicas = *svc.Spec.Replicas
	}
	modelImage := svc.Spec.Image
	if modelImage == "" {
		modelImage = backendImage
	}
	modelRepo := svc.Spec.Model.Repository
	if modelRepo == "" {
		modelRepo = "bartowski/SmolLM2-135M-Instruct-GGUF"
	}
	modelFile := svc.Spec.Model.File
	if modelFile == "" {
		modelFile = "SmolLM2-135M-Instruct-Q4_K_M.gguf"
	}
	modelRevision := svc.Spec.Model.Revision
	if modelRevision == "" {
		modelRevision = "main"
	}
	var err error
	deployment := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: svc.Name, Namespace: svc.Namespace}}
	_, err = controllerutil.CreateOrUpdate(ctx, r.Client, deployment, func() error {
		labels := map[string]string{"app": svc.Name}
		deployment.Labels = labels
		deployment.Spec.Replicas = &replicas
		deployment.Spec.Selector = &metav1.LabelSelector{MatchLabels: labels}
		deployment.Spec.Template.ObjectMeta.Labels = labels
		deployment.Spec.Template.ObjectMeta.Annotations = map[string]string{
			"demo.platform/model-repo":     modelRepo,
			"demo.platform/model-file":     modelFile,
			"demo.platform/model-revision": modelRevision,
		}
		deployment.Spec.Template.Spec.Containers = []corev1.Container{{
			Name:  "llama-server",
			Image: modelImage,
			Args: []string{
				"--host", "0.0.0.0",
				"--port", fmt.Sprintf("%d", backendListenPort),
				"--hf-repo", modelRepo,
				"--hf-file", modelFile,
			},
			Ports:     []corev1.ContainerPort{{ContainerPort: backendListenPort}},
			Resources: svc.Spec.Resources,
		}}
		return controllerutil.SetControllerReference(&svc, deployment, r.Scheme)
	})
	if err != nil {
		logger.Error(err, "reconciling backend deployment")
		return ctrl.Result{}, err
	}
	logger.Info("backend deployment reconciled", "image", modelImage, "modelRepository", modelRepo, "modelFile", modelFile)

	service := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: svc.Name, Namespace: svc.Namespace}}
	_, err = controllerutil.CreateOrUpdate(ctx, r.Client, service, func() error {
		service.Spec.Selector = map[string]string{"app": svc.Name}
		service.Spec.Ports = []corev1.ServicePort{{Port: 80, TargetPort: intstrFromInt(backendListenPort)}}
		return controllerutil.SetControllerReference(&svc, service, r.Scheme)
	})
	if err != nil {
		logger.Error(err, "reconciling backend service")
		return ctrl.Result{}, err
	}
	logger.Info("backend service reconciled")

	if svc.Spec.Ingress.Enabled {
		ingress := &networkingv1.Ingress{ObjectMeta: metav1.ObjectMeta{Name: svc.Name, Namespace: svc.Namespace}}
		_, err = controllerutil.CreateOrUpdate(ctx, r.Client, ingress, func() error {
			ingress.Spec.Rules = []networkingv1.IngressRule{{
				Host: svc.Spec.Ingress.Host,
				IngressRuleValue: networkingv1.IngressRuleValue{HTTP: &networkingv1.HTTPIngressRuleValue{Paths: []networkingv1.HTTPIngressPath{{
					Path:     "/",
					PathType: func() *networkingv1.PathType { p := networkingv1.PathTypePrefix; return &p }(),
					Backend:  networkingv1.IngressBackend{Service: &networkingv1.IngressServiceBackend{Name: service.Name, Port: networkingv1.ServiceBackendPort{Number: 80}}},
				}}}},
			}}
			return controllerutil.SetControllerReference(&svc, ingress, r.Scheme)
		})
		if err != nil {
			logger.Error(err, "reconciling backend ingress")
			return ctrl.Result{}, err
		}
		logger.Info("backend ingress reconciled", "host", svc.Spec.Ingress.Host)
	}

	frontendCatalog, err := r.serviceCatalog(ctx, svc.Namespace)
	if err != nil {
		logger.Error(err, "building frontend catalog")
		return ctrl.Result{}, err
	}
	logger.Info("catalog built", "services", len(frontendCatalog))
	if err := r.ensureFrontend(ctx, svc.Namespace, frontendCatalog); err != nil {
		logger.Error(err, "reconciling shared frontend")
		return ctrl.Result{}, err
	}
	logger.Info("shared frontend reconciled", "frontendURL", frontendURL())

	phase := "Pending"
	readyReplicas := int32(0)
	var currentDeployment appsv1.Deployment
	if err := r.Get(ctx, types.NamespacedName{Name: deployment.Name, Namespace: deployment.Namespace}, &currentDeployment); err == nil {
		readyReplicas = currentDeployment.Status.ReadyReplicas
		if readyReplicas >= replicas && replicas > 0 {
			phase = "Ready"
		}
	} else if !apierrors.IsNotFound(err) {
		return ctrl.Result{}, err
	}

	svc.Status.Phase = phase
	svc.Status.ReadyReplicas = readyReplicas
	svc.Status.BackendMode = "llama.cpp"
	svc.Status.BackendURL = backendURLFor(svc.Namespace, svc.Name)
	svc.Status.FrontendURL = frontendURL()
	svc.Status.LastReconcileTime = metav1.Now()
	if err := r.Status().Update(ctx, &svc); err != nil {
		logger.Error(err, "updating status")
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}
	logger.Info("status updated", "phase", svc.Status.Phase, "readyReplicas", svc.Status.ReadyReplicas, "backendURL", svc.Status.BackendURL)

	return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
}

func (r *TinyLLMServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&demov1alpha1.TinyLLMService{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&networkingv1.Ingress{}).
		Complete(r)
}

func intstrFromInt(v int) intstr.IntOrString {
	return intstr.FromInt(v)
}

func (r *TinyLLMServiceReconciler) serviceCatalog(ctx context.Context, namespace string) ([]map[string]string, error) {
	var list demov1alpha1.TinyLLMServiceList
	if err := r.List(ctx, &list, client.InNamespace(namespace)); err != nil {
		return nil, err
	}
	entries := make([]map[string]string, 0, len(list.Items))
	for _, item := range list.Items {
		entries = append(entries, map[string]string{
			"name":            item.Name,
			"namespace":       item.Namespace,
			"backendUrl":      backendURLFor(item.Namespace, item.Name),
			"promptPrefix":    item.Spec.PromptPrefix,
			"modelRepository": item.Spec.Model.Repository,
			"modelFile":       item.Spec.Model.File,
			"modelRevision":   item.Spec.Model.Revision,
		})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i]["name"] < entries[j]["name"] })
	return entries, nil
}

func (r *TinyLLMServiceReconciler) ensureFrontend(ctx context.Context, namespace string, services []map[string]string) error {
	configMap := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: frontendConfigMap, Namespace: namespace}}
	configJSON, err := json.Marshal(map[string]any{"services": services})
	if err != nil {
		return err
	}
	if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, configMap, func() error {
		configMap.Data = map[string]string{"services.json": string(configJSON)}
		return nil
	}); err != nil {
		return err
	}

	deployment := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: frontendName, Namespace: namespace}}
	if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, deployment, func() error {
		labels := map[string]string{"app": frontendName}
		deployment.Labels = labels
		deployment.Spec.Replicas = int32Ptr(1)
		deployment.Spec.Selector = &metav1.LabelSelector{MatchLabels: labels}
		deployment.Spec.Template.ObjectMeta.Labels = labels
		deployment.Spec.Template.Spec.Containers = []corev1.Container{{
			Name:         "frontend",
			Image:        frontendImage,
			Env:          []corev1.EnvVar{{Name: "CATALOG_PATH", Value: "/etc/tiny-llm/catalog/services.json"}},
			Ports:        []corev1.ContainerPort{{ContainerPort: frontendListenPort}},
			VolumeMounts: []corev1.VolumeMount{{Name: "catalog", MountPath: "/etc/tiny-llm/catalog", ReadOnly: true}},
		}}
		deployment.Spec.Template.Spec.Volumes = []corev1.Volume{{
			Name:         "catalog",
			VolumeSource: corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{LocalObjectReference: corev1.LocalObjectReference{Name: configMap.Name}}},
		}}
		return nil
	}); err != nil {
		return err
	}

	service := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: frontendName, Namespace: namespace}}
	if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, service, func() error {
		service.Spec.Selector = map[string]string{"app": frontendName}
		service.Spec.Ports = []corev1.ServicePort{{Port: 80, TargetPort: intstrFromInt(frontendListenPort)}}
		return nil
	}); err != nil {
		return err
	}

	ingress := &networkingv1.Ingress{ObjectMeta: metav1.ObjectMeta{Name: frontendName, Namespace: namespace}}
	if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, ingress, func() error {
		ingress.Spec.Rules = []networkingv1.IngressRule{{
			Host: frontendHost,
			IngressRuleValue: networkingv1.IngressRuleValue{HTTP: &networkingv1.HTTPIngressRuleValue{Paths: []networkingv1.HTTPIngressPath{{
				Path:     "/",
				PathType: func() *networkingv1.PathType { p := networkingv1.PathTypePrefix; return &p }(),
				Backend:  networkingv1.IngressBackend{Service: &networkingv1.IngressServiceBackend{Name: service.Name, Port: networkingv1.ServiceBackendPort{Number: 80}}},
			}}}},
		}}
		return nil
	}); err != nil {
		return err
	}

	return nil
}

func backendURLFor(namespace, name string) string {
	return fmt.Sprintf("http://%s.%s.svc.cluster.local", name, namespace)
}

func frontendURL() string {
	return fmt.Sprintf("https://%s", frontendHost)
}

func int32Ptr(v int32) *int32 {
	return &v
}
