package main

import (
	"flag"
	"os"

	demov1alpha1 "devops-demo/operator/api/v1alpha1"
	"devops-demo/operator/controllers"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

func main() {
	var metricsAddr string
	var probeAddr string
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))
	ctrl.Log.WithName("bootstrap").Info("starting operator", "metricsAddr", metricsAddr, "probeAddr", probeAddr)

	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(demov1alpha1.AddToScheme(scheme))
	utilruntime.Must(appsv1.AddToScheme(scheme))
	utilruntime.Must(corev1.AddToScheme(scheme))
	utilruntime.Must(networkingv1.AddToScheme(scheme))

	manager, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		Metrics:                metricsserver.Options{BindAddress: metricsAddr},
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         false,
	})
	if err != nil {
		ctrl.Log.WithName("bootstrap").Error(err, "creating manager")
		os.Exit(1)
	}

	if err := (&controllers.TinyLLMServiceReconciler{Client: manager.GetClient(), Scheme: scheme}).SetupWithManager(manager); err != nil {
		ctrl.Log.WithName("bootstrap").Error(err, "setting up controller")
		os.Exit(1)
	}

	if err := manager.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		ctrl.Log.WithName("bootstrap").Error(err, "adding healthz check")
		os.Exit(1)
	}
	if err := manager.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		ctrl.Log.WithName("bootstrap").Error(err, "adding readyz check")
		os.Exit(1)
	}

	ctrl.Log.WithName("bootstrap").Info("operator ready")
	if err := manager.Start(ctrl.SetupSignalHandler()); err != nil {
		ctrl.Log.WithName("bootstrap").Error(err, "starting manager")
		os.Exit(1)
	}
}
