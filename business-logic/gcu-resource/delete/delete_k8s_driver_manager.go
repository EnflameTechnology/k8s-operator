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

type DeleteK8sDriverManager struct {
	GcuResource *v1.GcuResource
	Client      client.Client
	Name        string
	UUID        string
}

var _ interfaces.Procedure = &DeleteK8sDriverManager{}

func NewDeleteK8sDriverManager(gcuResource *v1.GcuResource, client client.Client) interfaces.Procedure {
	return &DeleteK8sDriverManager{
		GcuResource: gcuResource,
		Client:      client,
		Name:        gcuResource.Name,
		UUID:        string(gcuResource.UID),
	}
}

func (rer *DeleteK8sDriverManager) GetProcedureName() string {
	return "DeleteK8sDriverManager"
}

func (rer *DeleteK8sDriverManager) GetProcedureStatus() []string {
	return []string{gcuStatus.DELETING, gcuStatus.DELETE_K8S_DRIVER_MANAGER_FAILED}
}

func (rer *DeleteK8sDriverManager) GetProcedureOKStatus() string {
	return gcuStatus.DELETE_K8S_DRIVER_MANAGER_SUCCESS
}

func (rer *DeleteK8sDriverManager) GetProcedureFailStatus() string {
	return gcuStatus.DELETE_K8S_DRIVER_MANAGER_FAILED
}

func (rer *DeleteK8sDriverManager) GetProcedureOKProgress() int {
	return gcuPro.DELETE_K8S_DRIVER_MANAGER_SUCCESS
}

func (rer *DeleteK8sDriverManager) GetProcedureFailProgress() int {
	return gcuPro.DELETE_K8S_DRIVER_MANAGER_FAILED
}

func (rer *DeleteK8sDriverManager) GetProcedureAction() string {
	return consts.DELETE
}

func (rer *DeleteK8sDriverManager) GetGCUResource() *v1.GcuResource {
	return rer.GcuResource
}

func (rer *DeleteK8sDriverManager) GetClient() client.Client {
	return rer.Client
}

func (rer *DeleteK8sDriverManager) Execute() (err error) {
	// logic here
	status := &base.ProcessStatus{Procedure: rer}
	defer status.FinallySetStatus(&err)
	if rer.GcuResource.Spec.K8sDriverManager == nil || (rer.GcuResource.Spec.K8sDriverManager.Deploy != nil &&
		!*rer.GcuResource.Spec.K8sDriverManager.Deploy) {
		logs.Info("resource %s need not delete, because it not deployed", consts.K8sDriverManager)
		return nil
	}
	status.ResourceStatus = &v1.ResourceStatus{
		Name:      consts.K8sDriverManager,
		Namespace: base.GetNamespace(consts.K8sDriverManager, rer.GcuResource.Spec.K8sDriverManager.Namespace),
		Status:    gcuStatus.DELETING,
	}
	resourceStatuses := rer.GcuResource.Status.ResourceStatuses
	for _, resourceStatus := range resourceStatuses {
		if resourceStatus.Name == consts.K8sDriverManager {
			resourceInfo := resourceStatus.OwnResources
			for _, resInfo := range resourceInfo {
				namespaceName := k8sTypes.NamespacedName{Namespace: resourceStatus.Namespace, Name: resInfo.Name}
				if err := base.DeleteClusterResource(rer.Client, resInfo, &namespaceName); err != nil {
					return err
				}
				logs.Info("delete cluster resource(kind:%s, name:%s, namespace:%s) of %s success",
					resInfo.Kind, resInfo.Name, resourceStatus.Namespace, consts.K8sDriverManager)
			}
		}
	}
	return nil
}
