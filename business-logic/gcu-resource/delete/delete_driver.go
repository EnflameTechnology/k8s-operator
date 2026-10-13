// Copyright (c) 2022 Enflame. All Rights Reserved.

package delete

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	k8sTypes "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "gcu-operator/apis/topscloud/v1"
	"gcu-operator/business-logic/base/interfaces"
	"gcu-operator/business-logic/gcu-resource/base"
	gcuPro "gcu-operator/business-logic/gcu-resource/base/progress"
	gcuStatus "gcu-operator/business-logic/gcu-resource/base/status"
	"gcu-operator/pkg/consts"
	gcuErr "gcu-operator/pkg/error"
	"gcu-operator/pkg/logs"
)

type DeleteDriver struct {
	GcuResource *v1.GcuResource
	Client      client.Client
	Name        string
	UUID        string
}

var _ interfaces.Procedure = &DeleteDriver{}

func NewDeleteDriver(gcuResource *v1.GcuResource, client client.Client) interfaces.Procedure {
	return &DeleteDriver{
		GcuResource: gcuResource,
		Client:      client,
		Name:        gcuResource.Name,
		UUID:        string(gcuResource.UID),
	}
}

func (rer *DeleteDriver) GetProcedureName() string {
	return "DeleteDriver"
}

func (rer *DeleteDriver) GetProcedureStatus() []string {
	return []string{gcuStatus.DELETE_TOOLKIT_SUCCESS, gcuStatus.DELETE_DRIVER_FAILED}
}

func (rer *DeleteDriver) GetProcedureOKStatus() string {
	return gcuStatus.DELETE_DRIVER_SUCCESS
}

func (rer *DeleteDriver) GetProcedureFailStatus() string {
	return gcuStatus.DELETE_DRIVER_FAILED
}

func (rer *DeleteDriver) GetProcedureOKProgress() int {
	return gcuPro.DELETE_DRIVER_SUCCESS
}

func (rer *DeleteDriver) GetProcedureFailProgress() int {
	return gcuPro.DELETE_DRIVER_FAILED
}

func (rer *DeleteDriver) GetProcedureAction() string {
	return consts.DELETE
}

func (rer *DeleteDriver) GetGCUResource() *v1.GcuResource {
	return rer.GcuResource
}

func (rer *DeleteDriver) GetClient() client.Client {
	return rer.Client
}

func (rer *DeleteDriver) Execute() (err error) {
	// logic here
	status := &base.ProcessStatus{Procedure: rer}
	defer status.FinallySetStatus(&err)
	if rer.GcuResource.Spec.Driver == nil || (rer.GcuResource.Spec.Driver.Deploy != nil && !*rer.GcuResource.Spec.Driver.Deploy) {
		logs.Info("resource %s need not delete, because it not deployed", consts.Driver)
		return nil
	}
	status.ResourceStatus = &v1.ResourceStatus{
		Name:      consts.Driver,
		Namespace: base.GetNamespace(consts.Driver, rer.GcuResource.Spec.Driver.Namespace),
		Status:    gcuStatus.DELETING,
	}
	resourceStatuses := rer.GcuResource.Status.ResourceStatuses
	for _, resourceStatus := range resourceStatuses {
		if resourceStatus.Name == consts.Driver {
			resourceInfo := resourceStatus.OwnResources
			for _, resInfo := range resourceInfo {
				namespaceName := k8sTypes.NamespacedName{Namespace: resourceStatus.Namespace, Name: resInfo.Name}
				if err := base.DeleteClusterResource(rer.Client, resInfo, &namespaceName); err != nil {
					return err
				}
				logs.Info("delete cluster resource(kind:%s, name:%s, namespace:%s) of %s success",
					resInfo.Kind, resInfo.Name, resourceStatus.Namespace, consts.Driver)
			}
		}
	}
	return rer.resetNodeLabels()
}

func (rer *DeleteDriver) resetNodeLabels() error {
	nodeList := &corev1.NodeList{}
	if err := rer.Client.List(context.TODO(), nodeList, &client.ListOptions{
		LabelSelector: labels.SelectorFromSet(map[string]string{
			consts.DriverInstalledLabel: "true",
		}),
	}); err != nil {
		logs.Error(err, "get cluster node list failed")
		return gcuErr.NewError(consts.ACTION_RETRY, err.Error())
	}
	for i := range nodeList.Items {
		node := &nodeList.Items[i]
		patch := client.MergeFrom(node.DeepCopy())
		delete(node.Labels, consts.DriverInstalledLabel)
		if err := rer.Client.Patch(context.TODO(), node, patch); err != nil {
			logs.Error(err, "remove driver installed label %s on node %s failed", consts.DriverInstalledLabel, node.Name)
			return gcuErr.NewError(consts.ACTION_RETRY, err.Error())
		}
		logs.Info("remove driver installed label %s on node %s success",
			consts.DriverInstalledLabel, node.Name)
	}
	return nil
}
