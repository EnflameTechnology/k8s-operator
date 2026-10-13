// Copyright (c) 2022 Enflame. All Rights Reserved.

package model

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	kubeErr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	gcuv1 "gcu-operator/apis/topscloud/v1"
	"gcu-operator/business-logic/base/executor"
	gcuPro "gcu-operator/business-logic/gcu-resource/base/progress"
	gcuStatus "gcu-operator/business-logic/gcu-resource/base/status"
	"gcu-operator/pkg/common"
	"gcu-operator/pkg/consts"
	gcuErr "gcu-operator/pkg/error"
	"gcu-operator/pkg/kube"
	"gcu-operator/pkg/logs"
)

type ModelGcuResouce struct {
	Ctx         context.Context
	Req         ctrl.Request
	Client      client.Client
	GcuInstance *gcuv1.GcuResource
}

func NewModel(ctx context.Context, req ctrl.Request, client client.Client) *ModelGcuResouce {
	return &ModelGcuResouce{
		Ctx:    ctx,
		Req:    req,
		Client: client,
	}
}

func (rer *ModelGcuResouce) Reconcile() (ctrl.Result, error) {
	defer func() {
		if err := recover(); err != nil {
			logs.Error(err, "internal error: handle reconciler message occurred panic")
		}
	}()
	// Fetch the GcuResource instance
	if err := rer.RefreshGcuResource(rer.Req.Name); err != nil {
		return reconcile.Result{RequeueAfter: time.Second * 2}, err
	}
	if rer.GcuInstance == nil {
		return reconcile.Result{}, nil
	}
	if action, err := rer.DoAction(); err != nil {
		if gcuErr, ok := err.(*gcuErr.Error); ok && gcuErr.Code == consts.ACTION_RETRY {
			logs.Warn("do action: %s failed: %s, need retry", action, gcuErr.Error())
		} else {
			logs.Error(err, "do action: %s failed, need retry", action)
		}
		return reconcile.Result{RequeueAfter: time.Second * 1}, err
	}
	return reconcile.Result{}, nil
}

func (rer *ModelGcuResouce) DoAction() (actionType string, err error) {
	switch rer.getActionType() {
	case consts.DEPLOY:
		err = rer.Deploy()
	case consts.DELETE:
		err = rer.Delete()
	case consts.UPGRADE:
		err = rer.Upgrade()
	case consts.ROLLBACK:
		err = rer.Rollback()
	}
	return
}

func (rer *ModelGcuResouce) getActionType() string {
	if rer.isDeleteAction() {
		return consts.DELETE
	}
	if rer.isDeployAction() {
		return consts.DEPLOY
	}
	if rer.isUpgradeAction() {
		return consts.UPGRADE
	}
	if rer.isRollbackction() {
		return consts.ROLLBACK
	}
	return ""
}

func (rer *ModelGcuResouce) isDeleteAction() bool {
	return rer.GcuInstance.DeletionTimestamp != nil
}

func (rer *ModelGcuResouce) isDeployAction() bool {
	status := rer.GcuInstance.Status
	if len(status.Conditions) == 0 || !rer.isDeployFinishStatus(status) {
		return true
	}
	return false
}

func (rer *ModelGcuResouce) isUpgradeAction() bool {
	// The upgrade action will be supported in the future
	return false
}

func (rer *ModelGcuResouce) isRollbackction() bool {
	// The rollback action will be supported in the future
	return false
}

func (rer *ModelGcuResouce) isDeployFinishStatus(status gcuv1.GcuResourceStatus) bool {
	conditions := status.Conditions
	actionCondition := common.FindConditionByActionType(consts.DEPLOY, conditions)
	return actionCondition.Status == gcuStatus.SUCCESS || actionCondition.Status == gcuStatus.FAILED
}

func (rer *ModelGcuResouce) Deploy() error {
	if err := rer.initDeployStatus(); err != nil {
		return err
	}
	executorIns := executor.NewExecutor(rer.GcuInstance, rer.Ctx, rer.Client, consts.DEPLOY)
	return rer.setDeployFinishStatus(executorIns.Execute())
}

func (rer *ModelGcuResouce) initDeployStatus() error {
	if len(rer.GcuInstance.Status.Conditions) != 0 {
		return nil
	}
	patch := client.MergeFrom(rer.GcuInstance.DeepCopy())
	if err := common.ReplaceVersion(rer.GcuInstance); err != nil {
		logs.Error(err, "replace gcu resource version failed")
		return err
	}
	// add finalizers
	if err := rer.addFinalizers(patch); err != nil {
		return err
	}
	instance := rer.GcuInstance
	patch = client.MergeFrom(instance.DeepCopy())
	// init deploy condition
	condition := gcuv1.Condition{
		Action:         consts.DEPLOY,
		Status:         gcuStatus.DEPLOYING,
		Reason:         gcuStatus.DEPLOYING,
		Message:        gcuStatus.DEPLOYING,
		Progress:       gcuPro.DEPLOYING,
		LastUpdateTime: &metav1.Time{Time: time.Now()},
	}
	instance.Status.Phase = consts.DEPLOYING
	instance.Status.ReadyResource = fmt.Sprintf("0/%d", rer.countResource())
	instance.Status.ObservedGeneration = 1
	instance.Status.Conditions = append(instance.Status.Conditions, condition)
	// err := rer.Client.Status().Update(context.TODO(), instance)
	err := rer.Client.Status().Patch(context.TODO(), instance, patch)
	if err != nil {
		logs.Error(err, "Failed to update gcuResource(name:%s, uuid:%s) status",
			instance.Name, instance.UID)
		return err
	}
	logs.Info("init gcuResource(name:%s, uuid:%s) deploy condition success", instance.Name, instance.UID)
	return nil
}

func (rer *ModelGcuResouce) addFinalizers(patch client.Patch) error {
	// add finalizers
	instance := rer.GcuInstance
	for _, finalizer := range instance.Finalizers {
		// finalizer already exist, can not add again
		if finalizer == consts.GCU_RESOURCE_INFO {
			return nil
		}
	}
	instance.Finalizers = append(instance.Finalizers, consts.GCU_RESOURCE_INFO)
	err := rer.Client.Patch(context.TODO(), instance, patch)
	if err != nil {
		logs.Error(err, "Failed to update gcuResource(name:%s, uuid:%s) finalizers",
			instance.Name, instance.UID)
		return err
	}
	logs.Info("add finalizers for gcuResource(name:%s, uuid:%s) success", instance.Name, instance.UID)
	return nil
}

func (rer *ModelGcuResouce) setDeployFinishStatus(errPro error) error {
	// if err := rer.RefreshGcuResource(); err != nil {
	// 	return err
	// }
	comOp := &common.OperateGCUResource{Client: rer.Client, GcuInstance: rer.GcuInstance}
	if errPro == nil {
		err := comOp.SetConditions(consts.RUNNING, consts.DEPLOY, gcuStatus.SUCCESS, gcuStatus.SUCCESS,
			gcuStatus.SUCCESS, gcuPro.SUCCESS)
		if err != nil {
			logs.Error(err, "set gcuResource(name:%s, uuid:%s) deploy success condition failed",
				rer.GcuInstance.Name, rer.GcuInstance.UID)
			return err
		}
		logs.Info("deploy gcuResource(name:%s, uuid:%s) success", rer.GcuInstance.Name, rer.GcuInstance.UID)
		// gcuInfo, err := common.ChangeInterfaceToString(rer.GcuInstance)
		// if err != nil {
		// 	return err
		// }
		// logs.Info("gcu resource detail after deploy success:%s", gcuInfo)
		return nil
	}
	gcuErr, ok := errPro.(*gcuErr.Error)
	if !ok {
		logs.Error(fmt.Errorf("the error of procedure must be gcuErr.Error"), "gcuResource(name:%s, uuid:%s)",
			rer.GcuInstance.Name, rer.GcuInstance.UID)
		return errPro
	}
	if gcuErr.Code == consts.ACTION_FAIL {
		err := comOp.SetConditions(consts.FAILED, consts.DEPLOY, gcuStatus.FAILED, gcuStatus.FAILED,
			gcuErr.Message, gcuPro.FAILED)
		if err != nil {
			logs.Error(err, "set gcuResource(name:%s, uuid:%s) deploy failed condition failed",
				rer.GcuInstance.Name, rer.GcuInstance.UID)
			return err
		}
		logs.Error(gcuErr, "deploy gcuResource(name:%s, uuid:%s) failed",
			rer.GcuInstance.Name, rer.GcuInstance.UID)
		return nil
	}
	if gcuErr.Code == consts.ACTION_RETRY {
		logs.Warn("deploy gcuResource(name:%s, uuid:%s) failed: %s, need retry",
			rer.GcuInstance.Name, rer.GcuInstance.UID, gcuErr.Error())
		return gcuErr
	}
	if gcuErr.Code == consts.ACTION_STOP {
		err := comOp.SetConditions(consts.RUNNING, consts.DEPLOY, gcuStatus.SUCCESS, gcuStatus.SUCCESS,
			gcuErr.Error(), gcuPro.SUCCESS)
		if err != nil {
			logs.Error(err, "set gcuResource(name:%s, uuid:%s) deploy success condition failed",
				rer.GcuInstance.Name, rer.GcuInstance.UID)
			return err
		}
		logs.Warn("deploy gcuResource(name:%s, uuid:%s) success, but %s", rer.GcuInstance.Name,
			rer.GcuInstance.UID, gcuErr.Error())
		logs.Info("gcu resource detail after deploy success:%s", common.ChangeInterfaceToString(rer.GcuInstance))
		return nil
	}
	logs.Error(gcuErr, "the error code of gcuResource(name:%s, uuid:%s) procedure must be 0 or 1, can not be:%d",
		rer.GcuInstance.Name, rer.GcuInstance.UID, gcuErr.Code)
	return gcuErr
}

func (rer *ModelGcuResouce) Delete() error {
	if err := rer.initDeleteStatus(); err != nil {
		return err
	}
	executorIns := executor.NewExecutor(rer.GcuInstance, rer.Ctx, rer.Client, consts.DELETE)
	if err := executorIns.Execute(); err != nil {
		if gcuErr, ok := err.(*gcuErr.Error); ok && gcuErr.Code == consts.ACTION_RETRY {
			logs.Warn("delete gcuResource(name:%s, uuid:%s) failed: %s, need retry",
				rer.GcuInstance.Name, rer.GcuInstance.UID, gcuErr.Error())
		} else {
			logs.Error(err, "delete gcuResource(name:%s, uuid:%s) failed, need retry",
				rer.GcuInstance.Name, rer.GcuInstance.UID)
		}
		return err
	}
	logs.Info("delete gcuResource(name:%s, uuid:%s) success", rer.GcuInstance.Name, rer.GcuInstance.UID)
	return nil
}

func (rer *ModelGcuResouce) initDeleteStatus() error {
	if common.CheckConditionExist(consts.DELETE, rer.GcuInstance.Status.Conditions) {
		return nil
	}
	// add finalizers
	instance := rer.GcuInstance
	patch := client.MergeFrom(instance.DeepCopy())
	// init delete condition
	condition := gcuv1.Condition{
		Action:         consts.DELETE,
		Status:         gcuStatus.DELETING,
		Reason:         gcuStatus.DELETING,
		Message:        gcuStatus.DELETING,
		Progress:       gcuPro.DELETING,
		LastUpdateTime: &metav1.Time{Time: time.Now()},
	}
	instance.Status.Phase = consts.DELETING
	instance.Status.Conditions = append(instance.Status.Conditions, condition)
	// err := rer.Client.Status().Update(context.TODO(), instance)
	err := rer.Client.Status().Patch(context.TODO(), instance, patch)
	if err != nil {
		logs.Error(err, "Failed to update gcuResource(name:%s, uuid:%s) status",
			instance.Name, instance.UID)
		return err
	}
	logs.Info("init gcuResource(name:%s, uuid:%s) delete condition success", instance.Name, instance.UID)
	// gcuInfo, err := common.ChangeInterfaceToString(rer.GcuInstance)
	// if err != nil {
	// 	return err
	// }
	// logs.Info("gcu resource detail after init:%s", gcuInfo)
	return nil
}

func (rer *ModelGcuResouce) Upgrade() error {
	// The upgrade action will be supported in the future
	return nil
}

func (rer *ModelGcuResouce) Rollback() error {
	// The rollback action will be supported in the future
	return nil
}

func (rer *ModelGcuResouce) RefreshGcuResource(name string) error {
	dynamicClient, err := kube.GetClientByGVK(consts.GROUP, consts.VERSION, consts.KIND, "")
	if err != nil {
		logs.Error(err, "get gcu resource dynamic client failed")
		return err
	}
	gcuInstance, err := dynamicClient.Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		if kubeErr.IsNotFound(err) {
			logs.Warn("cluster gcu resource not found when refresh models")
			return nil
		}
		logs.Error(err, "get gcu resource from cluster failed")
		return err
	}
	gcuInstanceByte, err := gcuInstance.MarshalJSON()
	if err != nil {
		logs.Error(err, "marshal gcu resource to byte failed")
		return err
	}
	instance := &gcuv1.GcuResource{}
	if err = json.Unmarshal(gcuInstanceByte, instance); err != nil {
		logs.Error(err, "unmarshal gcu resource to struct failed")
		return err
	}
	rer.GcuInstance = instance
	return nil
}

func (rer *ModelGcuResouce) countResource() (count int) {
	grSpec := rer.GcuInstance.Spec
	value := reflect.ValueOf(grSpec)
	for i := 0; i < value.NumField(); i++ {
		if value.Field(i).IsNil() {
			continue
		}
		deploy := value.Field(i).Elem().FieldByName("Deploy")
		if !deploy.IsNil() && !deploy.Elem().Bool() {
			continue
		}
		count++
	}
	return count
}
