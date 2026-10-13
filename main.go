// Copyright (c) 2022 Enflame. All Rights Reserved.

package main

import (
	"flag"
	"fmt"
	"os"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	topscloudv1 "gcu-operator/apis/topscloud/v1"
	topscloudcontrollers "gcu-operator/controllers/topscloud"
	"gcu-operator/pkg/consts"
	"gcu-operator/pkg/informer"
	"gcu-operator/pkg/logs"
	//+kubebuilder:scaffold:imports
)

var Version string

var (
	scheme = runtime.NewScheme()
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(topscloudv1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	var versionFlag bool
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.BoolVar(&versionFlag, "version", false, "enable version output")
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()
	if versionFlag {
		fmt.Printf("Gcu Operator version: %s\n", Version)
		return
	}

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "4d4668d3.enflame.com",
		// LeaderElectionReleaseOnCancel defines if the leader should step down voluntarily
		// when the Manager ends. This requires the binary to immediately end when the
		// Manager is stopped, otherwise, this setting is unsafe. Setting this significantly
		// speeds up voluntary leader transitions as the new leader don't have to wait
		// LeaseDuration time first.
		//
		// In the default scaffold provided, the program ends immediately after
		// the manager stops, so would be fine to enable this option. However,
		// if you are doing or is intended to do any operation such as perform cleanups
		// after the manager stops then its usage might be unsafe.
		// LeaderElectionReleaseOnCancel: true,
	})
	if err != nil {
		logs.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err = (&topscloudcontrollers.GcuResourceReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		logs.Error(err, "unable to create controller for %s", "GcuResource")
		os.Exit(1)
	}
	logs.Info("setup gcu resource manager success...")
	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		logs.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		logs.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	nodeInformer, err := informer.NewNodeController(mgr.GetClient())
	if err != nil {
		logs.Error(err, "unable new node informer")
		os.Exit(1)
	}
	go nodeInformer.Run()

	logs.Info("Starting Controller:%s, Group:%s, Version:%s, Kind:%s", consts.GCURESOURCE, consts.GROUP,
		consts.VERSION, consts.KIND)
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		logs.Error(err, "problem running manager")
		os.Exit(1)
	}
}
