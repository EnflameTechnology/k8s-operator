// Copyright (c) 2022 Enflame. All Rights Reserved.

package delete

import (
	k8sTypes "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "gcu-operator/apis/topscloud/v1"
	"gcu-operator/business-logic/base/interfaces"
	"gcu-operator/business-logic/gcu-resource/base"
	gcuPro "gcu-operator/business-logic/gcu-resource/base/progress"
	gcuStatus "gcu-operator/business-logic/gcu-resource/base/status"
	"gcu-operator/pkg/consts"
	"gcu-operator/pkg/logs"
)

type DeleteGcuShareScheduler struct {
	GcuResource *v1.GcuResource
	Client      client.Client
	Name        string
	UUID        string
}

var _ interfaces.Procedure = &DeleteGcuShareScheduler{}

func NewDeleteGcuShareScheduler(gcuResource *v1.GcuResource, client client.Client) interfaces.Procedure {
	return &DeleteGcuShareScheduler{
		GcuResource: gcuResource,
		Client:      client,
		Name:        gcuResource.Name,
		UUID:        string(gcuResource.UID),
	}
}

func (rer *DeleteGcuShareScheduler) GetProcedureName() string {
	return "DeleteGcuShareScheduler"
}

func (rer *DeleteGcuShareScheduler) GetProcedureStatus() []string {
	return []string{gcuStatus.DELETE_K8S_DRIVER_MANAGER_SUCCESS, gcuStatus.DELETE_GCUSHARE_SCHEDULER_FAILED}
}

func (rer *DeleteGcuShareScheduler) GetProcedureOKStatus() string {
	return gcuStatus.DELETE_GCUSHARE_SCHEDULER_SUCCESS
}

func (rer *DeleteGcuShareScheduler) GetProcedureFailStatus() string {
	return gcuStatus.DELETE_GCUSHARE_SCHEDULER_FAILED
}

func (rer *DeleteGcuShareScheduler) GetProcedureOKProgress() int {
	return gcuPro.DELETE_GCUSHARE_SCHEDULER_SUCCESS
}

func (rer *DeleteGcuShareScheduler) GetProcedureFailProgress() int {
	return gcuPro.DELETE_GCUSHARE_SCHEDULER_FAILED
}

func (rer *DeleteGcuShareScheduler) GetProcedureAction() string {
	return consts.DELETE
}

func (rer *DeleteGcuShareScheduler) GetGCUResource() *v1.GcuResource {
	return rer.GcuResource
}

func (rer *DeleteGcuShareScheduler) GetClient() client.Client {
	return rer.Client
}

func (rer *DeleteGcuShareScheduler) Execute() (err error) {
	// logic here
	status := &base.ProcessStatus{Procedure: rer}
	defer status.FinallySetStatus(&err)
	if rer.GcuResource.Spec.GCUShare == nil || (rer.GcuResource.Spec.GCUShare.Deploy != nil && !*rer.GcuResource.Spec.GCUShare.Deploy) {
		logs.Info("resource %s need not delete, because it not deployed", consts.GCUShareScheduler)
		return nil
	}
	status.ResourceStatus = &v1.ResourceStatus{
		Name:      consts.GCUShareScheduler,
		Namespace: base.GetNamespace(consts.GCUShareScheduler, rer.GcuResource.Spec.GCUShare.SchedulerPlugin.Namespace),
		Status:    gcuStatus.DELETING,
	}
	resourceStatuses := rer.GcuResource.Status.ResourceStatuses
	for _, resourceStatus := range resourceStatuses {
		if resourceStatus.Name == consts.GCUShareScheduler {
			resourceInfo := resourceStatus.OwnResources
			for _, resInfo := range resourceInfo {
				namespaceName := k8sTypes.NamespacedName{Namespace: resourceStatus.Namespace, Name: resInfo.Name}
				if err := base.DeleteClusterResource(rer.Client, resInfo, &namespaceName); err != nil {
					return err
				}
				logs.Info("delete cluster resource(kind:%s, name:%s, namespace:%s) of %s success",
					resInfo.Kind, resInfo.Name, resourceStatus.Namespace, consts.GCUShareScheduler)
			}
		}
	}
	return nil
}
