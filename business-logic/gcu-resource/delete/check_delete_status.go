// Copyright (c) 2022 Enflame. All Rights Reserved.

package delete

import (
	"context"
	"fmt"
	"time"

	"github.com/iancoleman/orderedmap"
	"github.com/imdario/mergo"
	kubeErr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sTypes "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "gcu-operator/apis/topscloud/v1"
	"gcu-operator/business-logic/base/interfaces"
	"gcu-operator/business-logic/gcu-resource/base"
	gcuPro "gcu-operator/business-logic/gcu-resource/base/progress"
	gcuStatus "gcu-operator/business-logic/gcu-resource/base/status"
	"gcu-operator/pkg/consts"
	gcuErr "gcu-operator/pkg/error"
	"gcu-operator/pkg/kube"
	"gcu-operator/pkg/logs"
	"gcu-operator/pkg/object"
)

type CheckDeleteStatus struct {
	GcuResource *v1.GcuResource
	Client      client.Client
	Name        string
	UUID        string
}

var _ interfaces.Procedure = &CheckDeleteStatus{}

type ClusterRes struct {
	v1.ResourceInfo
	Namespace string
}

func NewCheckDeleteStatus(gcuResource *v1.GcuResource, client client.Client) interfaces.Procedure {
	return &CheckDeleteStatus{
		GcuResource: gcuResource,
		Client:      client,
		Name:        gcuResource.Name,
		UUID:        string(gcuResource.UID),
	}
}

func (rer *CheckDeleteStatus) GetProcedureName() string {
	return "CheckDeleteStatus"
}

func (rer *CheckDeleteStatus) GetProcedureStatus() []string {
	return []string{gcuStatus.DELETE_NFD_SUCCESS, gcuStatus.CHECK_DELETE_STATUS_FAILED}
}

func (rer *CheckDeleteStatus) GetProcedureOKStatus() string {
	return gcuStatus.CHECK_DELETE_STATUS_FINISH
}

func (rer *CheckDeleteStatus) GetProcedureFailStatus() string {
	return gcuStatus.CHECK_DELETE_STATUS_FAILED
}

func (rer *CheckDeleteStatus) GetProcedureOKProgress() int {
	return gcuPro.CHECK_DELETE_STATUS_FINISH
}

func (rer *CheckDeleteStatus) GetProcedureFailProgress() int {
	return gcuPro.CHECK_DELETE_STATUS_FAILED
}

func (rer *CheckDeleteStatus) GetProcedureAction() string {
	return consts.DELETE
}

func (rer *CheckDeleteStatus) GetGCUResource() *v1.GcuResource {
	return rer.GcuResource
}

func (rer *CheckDeleteStatus) GetClient() client.Client {
	return rer.Client
}

func (rer *CheckDeleteStatus) Execute() (err error) {
	// logic here
	status := &base.ProcessStatus{Procedure: rer}
	defer status.FinallySetStatus(&err)
	resourceStatus := &[]v1.ResourceStatus{}
	defer rer.updateResourceDeleteStatus(resourceStatus, &err)
	componentResList := rer.getComponentResList()
	for _, component := range componentResList.Keys() {
		resList, _ := componentResList.Get(component)
		deleteFinish, err := rer.checkClusterResourceDeleteFinish(component, resList.([]ClusterRes))
		if err != nil {
			return err
		}
		if deleteFinish {
			*resourceStatus = append(*resourceStatus, v1.ResourceStatus{Name: component,
				Status: gcuStatus.SUB_RESOURCE_DELETED})
			logs.Info("check resource of %s delete finish", component)
		} else {
			logs.Warn("check resource of %s is deleting", component)
		}
	}
	if len(*resourceStatus) != len(componentResList.Keys()) {
		msg := fmt.Sprintf("check resouce delete not all finish, finish count:%d, all count:%d",
			len(*resourceStatus), len(componentResList.Keys()))
		logs.Warn(msg)
		time.Sleep(time.Second * consts.Interval)
		return gcuErr.NewError(consts.ACTION_RETRY, msg)
	}
	logs.Info("check delete all resource of gcu resource(name:%s, uuid:%s) finish", rer.Name, rer.UUID)
	return rer.removeFinalizers()
}

func (rer *CheckDeleteStatus) checkClusterResourceDeleteFinish(component string, resList []ClusterRes) (bool, error) {
	for _, resource := range resList {
		obj := object.Objects()[resource.Kind]
		namespacedName := k8sTypes.NamespacedName{Namespace: resource.Namespace, Name: resource.Name}
		err := rer.Client.Get(context.TODO(), namespacedName, obj)
		if err != nil {
			if kubeErr.IsNotFound(err) {
				logs.Info("check resource %v of component:%s delete finish", resource, component)
				continue
			}
			logs.Warn("use operator k8s client get %s(group: %s, version: %s) failed, error: %s", resource.Kind,
				resource.Group, resource.Version, err.Error())
			if err = rer.GetResourceUseDyClient(resource); err != nil {
				if kubeErr.IsNotFound(err) {
					logs.Info("check resource %v of component:%s delete finish", resource, component)
					continue
				}
				logs.Error(err, "get resource %v of component:%s from cluster failed",
					resource, component)
				return false, gcuErr.NewError(consts.ACTION_RETRY, err.Error())
			}
		}
		logs.Warn("check resource %v of component:%s is deleting", resource, component)
		return false, nil
	}
	return true, nil
}

func (rer *CheckDeleteStatus) GetResourceUseDyClient(resource ClusterRes) error {
	dynamicClient, err := kube.GetClientByGVK(resource.Group, resource.Version, resource.Kind, "")
	if err != nil {
		logs.Error(err, "get %s dynamic client failed", resource.Kind)
		return err
	}
	_, err = dynamicClient.Get(context.TODO(), resource.Name, metav1.GetOptions{})
	return err
}

func (rer *CheckDeleteStatus) removeFinalizers() error {
	instance := rer.GcuResource
	patch := client.MergeFrom(instance.DeepCopy())
	instance.Finalizers = []string{}
	err := rer.Client.Patch(context.TODO(), instance, patch)
	if err != nil {
		logs.Error(err, "Failed to update gcuResource(name:%s, uuid:%s) status",
			instance.Name, instance.UID)
		return err
	}
	return nil
}

func (rer *CheckDeleteStatus) updateResourceDeleteStatus(resourceStatus *[]v1.ResourceStatus, err *error) {
	// delete finish, need not update gcu resource
	if *err == nil {
		return
	}
	gcuResource := rer.GcuResource
	patch := client.MergeFrom(gcuResource.DeepCopy())
	for _, expectStatus := range *resourceStatus {
		for index, actualStatus := range gcuResource.Status.ResourceStatuses {
			if expectStatus.Name == actualStatus.Name {
				errIn := mergo.Merge(&actualStatus, expectStatus, mergo.WithOverride)
				if errIn != nil {
					*err = errIn
					logs.Error(errIn, "merge expect resource status:%v and actual resource status:%v failed",
						expectStatus, actualStatus)
					return
				}
				gcuResource.Status.ResourceStatuses[index] = actualStatus
				break
			}
		}
	}
	errUpdate := rer.Client.Status().Patch(context.TODO(), gcuResource, patch)
	if errUpdate != nil {
		logs.Error(errUpdate, "Failed to update gcuResource(name:%s, uuid:%s) status",
			gcuResource.Name, gcuResource.UID)
		*err = gcuErr.NewError(consts.ACTION_RETRY, errUpdate.Error())
		return
	}
}

func (rer *CheckDeleteStatus) getComponentResList() *orderedmap.OrderedMap {
	resourceStatuses := rer.GcuResource.Status.ResourceStatuses

	componentResList := orderedmap.New()
	componentResList.Set(consts.K8sDriverManager, rer.getComponentOwnResource(resourceStatuses, consts.K8sDriverManager))
	componentResList.Set(consts.GCUShareScheduler, rer.getComponentOwnResource(resourceStatuses, consts.GCUShareScheduler))
	componentResList.Set(consts.GCUSharePlugin, rer.getComponentOwnResource(resourceStatuses, consts.GCUSharePlugin))
	componentResList.Set(consts.GFD, rer.getComponentOwnResource(resourceStatuses, consts.GFD))
	componentResList.Set(consts.GcuExporter, rer.getComponentOwnResource(resourceStatuses, consts.GcuExporter))
	componentResList.Set(consts.NodeExporter, rer.getComponentOwnResource(resourceStatuses, consts.NodeExporter))
	componentResList.Set(consts.Plugin, rer.getComponentOwnResource(resourceStatuses, consts.Plugin))
	componentResList.Set(consts.Toolkit, rer.getComponentOwnResource(resourceStatuses, consts.Toolkit))
	componentResList.Set(consts.Driver, rer.getComponentOwnResource(resourceStatuses, consts.Driver))
	componentResList.Set(consts.NFD, rer.getComponentOwnResource(resourceStatuses, consts.NFD))
	return componentResList
}

func (rer *CheckDeleteStatus) getComponentOwnResource(resourceStatuses []v1.ResourceStatus,
	componentName string) (clusterRes []ClusterRes) {
	for _, resourceStatus := range resourceStatuses {
		if resourceStatus.Name == componentName {
			return rer.addNamespaceForClusterRes(resourceStatus.Namespace, resourceStatus.OwnResources)
		}
	}
	return clusterRes
}

func (rer *CheckDeleteStatus) addNamespaceForClusterRes(namespace string, ownResources []v1.ResourceInfo) (clusterRes []ClusterRes) {
	for _, ownResource := range ownResources {
		clusterRes = append(clusterRes, ClusterRes{
			Namespace:    namespace,
			ResourceInfo: ownResource,
		})
	}
	return clusterRes
}
