package controllers

import (
	"context"
	"fmt"
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

type TinyLLMServiceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *TinyLLMServiceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var svc demov1alpha1.TinyLLMService
	if err := r.Get(ctx, req.NamespacedName, &svc); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	replicas := int32(1)
	if svc.Spec.Replicas != nil {
		replicas = *svc.Spec.Replicas
	}
	mode := svc.Spec.ModelMode
	if mode == "" {
		mode = "mock"
	}
	prefix := svc.Spec.PromptPrefix
	if prefix == "" {
		prefix = "Demo:"
	}

	configMap := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: svc.Name, Namespace: svc.Namespace}}
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, configMap, func() error {
		configMap.Data = map[string]string{
			"MODEL_MODE":    mode,
			"PROMPT_PREFIX": prefix,
		}
		return controllerutil.SetControllerReference(&svc, configMap, r.Scheme)
	})
	if err != nil {
		return ctrl.Result{}, err
	}

	deployment := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: svc.Name, Namespace: svc.Namespace}}
	_, err = controllerutil.CreateOrUpdate(ctx, r.Client, deployment, func() error {
		labels := map[string]string{"app": svc.Name}
		deployment.Labels = labels
		deployment.Spec.Replicas = &replicas
		deployment.Spec.Selector = &metav1.LabelSelector{MatchLabels: labels}
		deployment.Spec.Template.Labels = labels
		deployment.Spec.Template.Annotations = map[string]string{"demo.platform/model-mode": mode}
		deployment.Spec.Template.Spec.Containers = []corev1.Container{{
			Name:  "tiny-llm",
			Image: svc.Spec.Image,
			EnvFrom: []corev1.EnvFromSource{{
				ConfigMapRef: &corev1.ConfigMapEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: configMap.Name}},
			}},
			Ports:     []corev1.ContainerPort{{ContainerPort: 8080}},
			Resources: svc.Spec.Resources,
		}}
		return controllerutil.SetControllerReference(&svc, deployment, r.Scheme)
	})
	if err != nil {
		return ctrl.Result{}, err
	}

	service := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: svc.Name, Namespace: svc.Namespace}}
	_, err = controllerutil.CreateOrUpdate(ctx, r.Client, service, func() error {
		service.Spec.Selector = map[string]string{"app": svc.Name}
		service.Spec.Ports = []corev1.ServicePort{{Port: 80, TargetPort: intstrFromInt(8080)}}
		return controllerutil.SetControllerReference(&svc, service, r.Scheme)
	})
	if err != nil {
		return ctrl.Result{}, err
	}

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
			return ctrl.Result{}, err
		}
	}

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
	svc.Status.BackendMode = mode
	svc.Status.URL = urlFor(svc.Namespace, svc.Name, svc.Spec.Ingress.Host)
	svc.Status.LastReconcileTime = metav1.Now()
	if err := r.Status().Update(ctx, &svc); err != nil {
		logger.Error(err, "updating status")
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}

	return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
}

func (r *TinyLLMServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&demov1alpha1.TinyLLMService{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&networkingv1.Ingress{}).
		Complete(r)
}

func intstrFromInt(v int) intstr.IntOrString {
	return intstr.FromInt(v)
}

func urlFor(namespace, name, host string) string {
	if host != "" {
		return fmt.Sprintf("https://%s", host)
	}
	return fmt.Sprintf("http://%s.%s.svc.cluster.local", name, namespace)
}
