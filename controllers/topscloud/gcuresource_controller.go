// Copyright (c) 2022 Enflame. All Rights Reserved.

package topscloud

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	topscloudv1 "gcu-operator/apis/topscloud/v1"
	"gcu-operator/model"
)

// GcuResourceReconciler reconciles a GcuResource object
type GcuResourceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=topscloud.enflame.com,resources=gcuresources,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=topscloud.enflame.com,resources=gcuresources/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=topscloud.enflame.com,resources=gcuresources/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the GcuResource object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.1/pkg/reconcile
func (r *GcuResourceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	// TODO(user): your logic here
	return model.NewModel(ctx, req, r.Client).Reconcile()
}

// SetupWithManager sets up the controller with the Manager.
func (r *GcuResourceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&topscloudv1.GcuResource{}).
		Complete(r)
}
