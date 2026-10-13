// Copyright (c) 2022 Enflame. All Rights Reserved.

package common

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"

	gcuv1 "gcu-operator/apis/topscloud/v1"
	"gcu-operator/pkg/consts"
	"gcu-operator/pkg/kube"
	"gcu-operator/pkg/logs"
)

func FindConditionByActionType(action string, conditions []gcuv1.Condition) gcuv1.Condition {
	for index := 0; index < len(conditions); index++ {
		if conditions[index].Action == action {
			return conditions[index]
		}
	}
	errMsg := fmt.Sprintf("can not get deploy condition by action:%s, conditions:%v", action, conditions)
	logs.Error(fmt.Errorf(errMsg))
	return gcuv1.Condition{}
}

func CheckConditionExist(action string, conditions []gcuv1.Condition) bool {
	for _, condition := range conditions {
		if condition.Action == action {
			return true
		}
	}
	return false
}

func ChangeInterfaceToString(value interface{}) string {
	byteValue, err := json.Marshal(value)
	if err != nil {
		logs.Error(err, "change interface to string failed")
		return fmt.Sprintf("%v", value)
	}
	return string(byteValue)
}

type OperateGCUResource struct {
	Client      client.Client
	GcuInstance *gcuv1.GcuResource
}

func (rer *OperateGCUResource) SetConditions(phase, action, status, reason, message string,
	progress int) error {
	gcuInstance := rer.GcuInstance
	patch := client.MergeFrom(gcuInstance.DeepCopy())
	gcuInstance.Status.Phase = phase
	conditions := gcuInstance.Status.Conditions

	condition := FindConditionByActionType(action, conditions)
	if condition.Action == "" {
		msg := fmt.Sprintf("can not find condition by action:%s", action)
		return fmt.Errorf(msg)
	}
	condition.Status = status
	condition.Reason = reason
	condition.Message = message
	condition.Progress = progress
	condition.LastUpdateTime = &metav1.Time{Time: time.Now()}
	for index := 0; index < len(conditions); index++ {
		if conditions[index].Action == action {
			conditions[index] = condition
		}
	}
	gcuInstance.Status.Conditions = conditions
	//err := rer.Client.Status().Update(context.TODO(), gcuInstance)
	err := rer.Client.Status().Patch(context.TODO(), gcuInstance, patch)
	if err != nil {
		logs.Error(err, "Failed to update gcuResource(name:%s, uuid:%s) status",
			gcuInstance.Name, gcuInstance.UID)
		return err
	}
	return nil
}

func CheckNFDRunning() error {
	labelSelector := map[string]string{
		consts.LABEL_INSTANCE: consts.OPERATOR_NAME,
		consts.LABEL_NAME:     consts.NFD,
	}
	maxRetryTimes := 100
	for retryTime := 0; retryTime < maxRetryTimes; retryTime++ {
		if checkNFDRunning(labelSelector) {
			logs.Info("node feature discovery start running")
			return nil
		}
		logs.Warn("check node feature discovery not running, retry times:%d", retryTime)
		time.Sleep(time.Second * consts.Interval)
	}
	return fmt.Errorf("check node feature discovery not running in %d seconds, operator will restart",
		maxRetryTimes*consts.Interval)
}

func checkNFDRunning(labelSelector map[string]string) bool {
	dynamicClient, err := kube.GetClientByGVK("apps", "v1", "DaemonSet", "")
	if err != nil {
		logs.Error(err, "get daemonset dynamic client failed")
		return false
	}
	opts := metav1.ListOptions{LabelSelector: labels.SelectorFromSet(labels.Set(labelSelector)).String()}
	daemonsetList, err := dynamicClient.List(context.TODO(), opts)
	if err != nil {
		logs.Error(err, "list cluster daemonset failed")
		return false
	}
	if len(daemonsetList.Items) != 1 {
		logs.Error(fmt.Errorf("daemonset with nfd labels more than once"), "daemonset list:%v", daemonsetList.Items)
		return false
	}
	daemonByte, err := daemonsetList.Items[0].MarshalJSON()
	if err != nil {
		logs.Error(err, "mashal daemonset unstructured to json failed")
		return false
	}
	daemonStruct := &appsv1.DaemonSet{}
	if err = json.Unmarshal(daemonByte, daemonStruct); err != nil {
		logs.Error(err, "unmashal daemonset json to struct failed")
		return false
	}
	if daemonStruct.Status.NumberReady == daemonStruct.Status.DesiredNumberScheduled {
		logs.Info("check node feature discovery is running")
		return true
	}
	// logs.Warn("check node feature discovery is not running")
	return false
}

// func checkNFDMasterRunning(labelSelector map[string]string) bool {
// 	dynamicClient, err := kube.GetClientByGVK("apps", "v1", "Deployment", "")
// 	if err != nil {
// 		logs.Error(err, "get deployment dynamic client failed")
// 		return false
// 	}
// 	opts := metav1.ListOptions{LabelSelector: labels.SelectorFromSet(labels.Set(labelSelector)).String()}
// 	deploymentList, err := dynamicClient.List(context.TODO(), opts)
// 	if err != nil {
// 		logs.Error(err, "list cluster deployment failed")
// 		return false
// 	}
// 	if len(deploymentList.Items) != 1 {
// 		logs.Error(fmt.Errorf("deployment with nfd master labels more than once"), "deployment list:%v", deploymentList.Items)
// 		return false
// 	}
// 	deployByte, err := deploymentList.Items[0].MarshalJSON()
// 	if err != nil {
// 		logs.Error(err, "mashal deployment unstructured to json failed")
// 		return false
// 	}
// 	deployStruct := &appsv1.Deployment{}
// 	if err = json.Unmarshal(deployByte, deployStruct); err != nil {
// 		logs.Error(err, "unmashal deployment json to struct failed")
// 		return false
// 	}
// 	if deployStruct.Status.ReadyReplicas == *deployStruct.Spec.Replicas {
// 		logs.Info("check node feature discovery master is running")
// 		return true
// 	}
// 	logs.Warn("check node feature discovery master is not running")
// 	return false
// }

func RemoveValue(value string, list []string) []string {
	res := []string{}
	for _, eachValue := range list {
		if eachValue != value {
			res = append(res, eachValue)
		}
	}
	return res
}

func HasValue(value string, list []string) bool {
	for _, eachValue := range list {
		if value == eachValue {
			return true
		}
	}
	return false
}

func FileIsExist(file string) bool {
	_, err := os.Stat(file)
	return err == nil || os.IsExist(err)
}

func GetVersionMap(fileName string) (map[string]string, error) {
	versionMap := make(map[string]string)
	file, err := os.Open(fileName)
	if err != nil {
		logs.Error(err, "open file %s failed", fileName)
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			name := strings.TrimSpace(parts[0])
			version := strings.TrimSpace(parts[1])
			versionMap[name] = version
		}
	}
	if err := scanner.Err(); err != nil {
		logs.Error(err, "read file %s failed", fileName)
		return nil, err
	}
	return versionMap, nil
}

func ReplaceVersion(gcuRes *gcuv1.GcuResource) error {
	versionMap, err := GetVersionMap(consts.VersionFile)
	if err != nil {
		return err
	}
	if gcuRes.Spec.Nfd != nil {
		if needReplace(gcuRes.Spec.Nfd.Master.Version) {
			gcuRes.Spec.Nfd.Master.Version = versionMap[consts.NODE_FEATURE_DISCOVERY_VERSION]
		}
		if needReplace(gcuRes.Spec.Nfd.Worker.Version) {
			gcuRes.Spec.Nfd.Worker.Version = versionMap[consts.NODE_FEATURE_DISCOVERY_VERSION]
		}
	}
	if gcuRes.Spec.Driver != nil && needReplace(gcuRes.Spec.Driver.Version) {
		gcuRes.Spec.Driver.Version = versionMap[consts.GCU_OPERATOR_DRIVER_VERSION]
	}
	if gcuRes.Spec.ContainerToolkit != nil && needReplace(gcuRes.Spec.ContainerToolkit.Version) {
		gcuRes.Spec.ContainerToolkit.Version = versionMap[consts.ENFLAME_CONTAINER_TOOLKIT_VERSION]
	}
	if gcuRes.Spec.DevicePlugin != nil && needReplace(gcuRes.Spec.DevicePlugin.Version) {
		gcuRes.Spec.DevicePlugin.Version = versionMap[consts.ENFLAME_DEVICE_PLUGIN_VERSION]
	}
	if gcuRes.Spec.GcuExporter != nil && needReplace(gcuRes.Spec.GcuExporter.Version) {
		gcuRes.Spec.GcuExporter.Version = versionMap[consts.GCU_EXPORTER_VERSION]
	}
	if gcuRes.Spec.NodeExporter != nil && needReplace(gcuRes.Spec.NodeExporter.Version) {
		gcuRes.Spec.NodeExporter.Version = versionMap[consts.NODE_EXPORTER_VERSION]
	}
	if gcuRes.Spec.GCUShare != nil && needReplace(gcuRes.Spec.GCUShare.DevicePlugin.Version) {
		gcuRes.Spec.GCUShare.DevicePlugin.Version = versionMap[consts.GCUSHARE_VERSION]
	}
	if gcuRes.Spec.GCUShare != nil && needReplace(gcuRes.Spec.GCUShare.SchedulerPlugin.Version) {
		gcuRes.Spec.GCUShare.SchedulerPlugin.Version = versionMap[consts.GCUSHARE_VERSION]
	}
	if gcuRes.Spec.Gfd != nil && needReplace(gcuRes.Spec.Gfd.Version) {
		gcuRes.Spec.Gfd.Version = versionMap[consts.GCU_FEATURE_DISCOVERY_VERSION]
	}
	if gcuRes.Spec.K8sDriverManager != nil && needReplace(gcuRes.Spec.K8sDriverManager.Version) {
		gcuRes.Spec.K8sDriverManager.Version = versionMap[consts.K8S_DRIVER_MANAGER_VERSION]
	}
	return nil
}

func needReplace(version string) bool {
	return version == "" || version == "default"
}
