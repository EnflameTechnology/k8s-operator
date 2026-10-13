// Copyright (c) 2022 Enflame. All Rights Reserved.

package base

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Masterminds/semver"
	"github.com/ghodss/yaml"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	kubeErr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8sTypes "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "gcu-operator/apis/topscloud/v1"
	"gcu-operator/business-logic/base/interfaces"
	gcuStatus "gcu-operator/business-logic/gcu-resource/base/status"
	"gcu-operator/pkg/common"
	"gcu-operator/pkg/consts"
	gcuErr "gcu-operator/pkg/error"
	"gcu-operator/pkg/kube"
	"gcu-operator/pkg/logs"
	"gcu-operator/pkg/object"
)

type ProcessStatus struct {
	// Required, Procedure define procedure object
	Procedure interfaces.Procedure

	// Optional, ObservedGeneration define the generation of gcu-resource
	ObservedGeneration int

	// Optional, IsReady define whether the current component is ready
	IsReady bool

	// Optional, AddNodesInfo define whether add nodes information to gcu resource status
	AddNodesInfo bool

	// Optional, ResourceStatus define the status of the current component
	ResourceStatus *v1.ResourceStatus
}

const (
	// gcu resource file path
	// gcu driver
	GcuDriverPath = "/opt/gcu-operator/driver"

	// gcu docker plugin
	GcuDockerPluginPath = "/opt/gcu-operator/container-toolkit"

	// gcu k8s plugin
	// v1
	GcuK8sPluginV1Path              = "/opt/gcu-operator/k8s-device-plugin/v1/base"
	GcuK8sPluginV1PathForPcieSwitch = "/opt/gcu-operator/k8s-device-plugin/v1/pcie-switch"
	// v2
	GcuK8sPluginV2Path = "/opt/gcu-operator/k8s-device-plugin/v2"

	// gcu exporter
	GcuExporterPath = "/opt/gcu-operator/gcu-exporter"

	// node exporter
	NodeExporterPath = "/opt/gcu-operator/node-exporter"

	// gcu k8s plugin
	GcuFeatureDiscoveryPath = "/opt/gcu-operator/gcu-feature-discovery"

	// nfd
	NfdPath = "/opt/gcu-operator/node-feature-discovery"

	// gcushare-device-plugin
	GcuSharePluginPath = "/opt/gcu-operator/gcushare-device-plugin"

	// gcushare-scheduler-plugin
	GcuShareSchedulerPath = "/opt/gcu-operator/gcushare-scheduler-plugin"

	// k8s-driver-manager
	K8sDriverManagerPath = "/opt/gcu-operator/k8s-driver-manager"
)

// FinallySetStatus used to update status fields but status.phase
func (ps *ProcessStatus) FinallySetStatus(err *error) {
	action := ps.Procedure.GetProcedureAction()
	if !common.HasValue(action, []string{consts.DEPLOY, consts.DELETE, consts.UPGRADE, consts.ROLLBACK}) {
		msg := fmt.Sprintf("invalid action: %s", action)
		*err = gcuErr.NewError(consts.ACTION_FAIL, msg)
		logs.Error(msg)
		return
	}
	if ps.checkGcuResourceHasDeleted(*err, action) {
		return
	}
	patch := client.MergeFrom(ps.Procedure.GetGCUResource().DeepCopy())
	ps.addNodesInfo(*err)
	ps.updateObservedGeneration()
	ps.updateReadyResource()
	ps.updateResourceStatus(*err)
	ps.updateCondition(*err)
	errUpdate := ps.Procedure.GetClient().Status().Patch(context.TODO(), ps.Procedure.GetGCUResource(), patch)
	if errUpdate != nil {
		logs.Error(errUpdate, "Failed to update gcuResource(name:%s, uuid:%s) status",
			ps.Procedure.GetGCUResource().Name, ps.Procedure.GetGCUResource().UID)
		*err = gcuErr.NewError(consts.ACTION_RETRY, errUpdate.Error())
		return
	}
}

func (ps *ProcessStatus) addNodesInfo(err error) {
	if err != nil || !ps.AddNodesInfo {
		return
	}
	nodeList := &corev1.NodeList{}
	for {
		errlist := ps.Procedure.GetClient().List(context.TODO(), nodeList, &client.ListOptions{})
		if errlist == nil {
			break
		}
		logs.Error(errlist, "get cluster node list failed")
		time.Sleep(time.Second * consts.Interval)
	}
	if ps.Procedure.GetGCUResource().Status.NodesInfo == nil {
		ps.Procedure.GetGCUResource().Status.NodesInfo = make(map[string]v1.NodeInfo, len(nodeList.Items))
	}
	for _, node := range nodeList.Items {
		_, ok := ps.Procedure.GetGCUResource().Status.NodesInfo[node.Name]
		if ok {
			logs.Info("node: %s information already exist in gcuresource(name: %s, uuid: %s), need not add",
				node.Name, ps.Procedure.GetGCUResource().Name, ps.Procedure.GetGCUResource().UID)
			continue
		}
		logs.Info("add node: %s information to gcuresource(name: %s, uuid: %s)", node.Name,
			ps.Procedure.GetGCUResource().Name, ps.Procedure.GetGCUResource().UID)
		_, ok = node.Labels[consts.UnifiedDeviceLabelKey]
		ps.Procedure.GetGCUResource().Status.NodesInfo[node.Name] = v1.NodeInfo{
			IsGCUNode:               ok,
			KubeletVersion:          node.Status.NodeInfo.KubeletVersion,
			ContainerRuntimeVersion: node.Status.NodeInfo.ContainerRuntimeVersion,
		}
	}
}

func (ps *ProcessStatus) updateCondition(err error) {
	var status, reason, message string
	var progress int
	if err == nil {
		status = ps.Procedure.GetProcedureOKStatus()
		reason = ps.Procedure.GetProcedureOKStatus()
		message = ps.Procedure.GetProcedureOKStatus()
		progress = ps.Procedure.GetProcedureOKProgress()
	} else {
		status = ps.Procedure.GetProcedureFailStatus()
		reason = ps.Procedure.GetProcedureFailStatus()
		message = err.Error()
		progress = ps.Procedure.GetProcedureFailProgress()
	}

	condition := v1.Condition{
		Action:         ps.Procedure.GetProcedureAction(),
		Status:         status,
		Reason:         reason,
		Message:        message,
		Progress:       progress,
		LastUpdateTime: &metav1.Time{Time: time.Now()},
	}
	for index, condi := range ps.Procedure.GetGCUResource().Status.Conditions {
		if condi.Action == ps.Procedure.GetProcedureAction() {
			ps.Procedure.GetGCUResource().Status.Conditions[index] = condition
		}
	}
}

func (ps *ProcessStatus) updateResourceStatus(err error) {
	if ps.ResourceStatus == nil {
		return
	}
	action := ps.Procedure.GetProcedureAction()
	if action == consts.DEPLOY {
		ps.updateDeployResourceStatus(err)
	} else if action == consts.DELETE {
		ps.updateDeleteResourceStatus()
	} else if action == consts.UPGRADE {
		ps.updateUpgradeResourceStatus()
	} else if action == consts.ROLLBACK {
		ps.updateRollbackResourceStatus()
	}
}

func (ps *ProcessStatus) updateDeployResourceStatus(err error) {
	if err == nil {
		ps.ResourceStatus.Status = gcuStatus.SUB_RESOURCE_RUNNING
	}
	for index, res := range ps.Procedure.GetGCUResource().Status.ResourceStatuses {
		if res.Name == ps.ResourceStatus.Name && res.Namespace == ps.ResourceStatus.Namespace {
			ps.Procedure.GetGCUResource().Status.ResourceStatuses[index] = *ps.ResourceStatus
			return
		}
	}
	ps.Procedure.GetGCUResource().Status.ResourceStatuses = append(
		ps.Procedure.GetGCUResource().Status.ResourceStatuses, *ps.ResourceStatus)
}

func (ps *ProcessStatus) updateDeleteResourceStatus() {
	for index, res := range ps.Procedure.GetGCUResource().Status.ResourceStatuses {
		if res.Name == ps.ResourceStatus.Name && res.Namespace == ps.ResourceStatus.Namespace {
			ps.Procedure.GetGCUResource().Status.ResourceStatuses[index].Status = ps.ResourceStatus.Status
		}
	}
}

func (ps *ProcessStatus) updateUpgradeResourceStatus() {
	// upgrade and rollback are not considered for the time being
}

func (ps *ProcessStatus) updateRollbackResourceStatus() {
	// upgrade and rollback are not considered for the time being
}

func (ps *ProcessStatus) updateObservedGeneration() {
	if ps.ObservedGeneration > 0 {
		ps.Procedure.GetGCUResource().Status.ObservedGeneration = ps.ObservedGeneration
	}
}

func (ps *ProcessStatus) updateReadyResource() {
	if !ps.IsReady {
		return
	}
	countList := strings.Split(ps.Procedure.GetGCUResource().Status.ReadyResource, "/")
	oldReadyCount := countList[0]
	newReadyCount, _ := strconv.Atoi(oldReadyCount)
	newReadyCount++
	countList[0] = strconv.Itoa(newReadyCount)
	ps.Procedure.GetGCUResource().Status.ReadyResource = strings.Join(countList, "/")
}

func ReadYamlToJson(filePath string) (string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", gcuErr.NewError(consts.ACTION_FAIL, err.Error())
	}
	jsonContent, err := yaml.YAMLToJSON(content)
	if err != nil {
		return "", gcuErr.NewError(consts.ACTION_FAIL, err.Error())
	}
	return string(jsonContent), nil
}

func AddDsNodeSelector(resourceStruct *appsv1.DaemonSet, nodeSelector map[string]string) {
	if resourceStruct.Spec.Template.Spec.NodeSelector == nil {
		resourceStruct.Spec.Template.Spec.NodeSelector = map[string]string{}
	}
	for key, value := range nodeSelector {
		resourceStruct.Spec.Template.Spec.NodeSelector[key] = value
	}
}

func AddDsEnv(resourceStruct *appsv1.DaemonSet, envs []corev1.EnvVar) {
	container := resourceStruct.Spec.Template.Spec.Containers[0]
	for _, env := range envs {
		if index := EnvExist(container.Env, env); index >= 0 {
			logs.Warn("env: %s already exist in container: %s, value: %s will replaced by: %s", env.Name,
				container.Name, container.Env[index].Value, env.Value)
			resourceStruct.Spec.Template.Spec.Containers[0].Env[index].Value = env.Value
		} else {
			logs.Info("add env: %v for container: %s", env, container.Name)
			resourceStruct.Spec.Template.Spec.Containers[0].Env = append(resourceStruct.Spec.Template.Spec.Containers[0].Env, env)
		}
	}
}

func EnvExist(envs []corev1.EnvVar, env corev1.EnvVar) int {
	for index, each := range envs {
		if each.Name == env.Name {
			return index
		}
	}
	return -1
}

func AddDeployNodeSelector(resourceStruct *appsv1.Deployment, nodeSelector map[string]string) {
	if resourceStruct.Spec.Template.Spec.NodeSelector == nil {
		resourceStruct.Spec.Template.Spec.NodeSelector = map[string]string{}
	}
	for key, value := range nodeSelector {
		resourceStruct.Spec.Template.Spec.NodeSelector[key] = value
	}
}

func AddDeployEnv(resourceStruct *appsv1.Deployment, envs []corev1.EnvVar) {
	container := resourceStruct.Spec.Template.Spec.Containers[0]
	for _, env := range envs {
		if index := EnvExist(container.Env, env); index >= 0 {
			logs.Warn("env: %s already exist in container: %s, value: %s will replaced by: %s", env.Name,
				container.Name, container.Env[index].Value, env.Value)
			resourceStruct.Spec.Template.Spec.Containers[0].Env[index].Value = env.Value
		} else {
			logs.Info("add env: %v for container: %s", env, container.Name)
			resourceStruct.Spec.Template.Spec.Containers[0].Env = append(resourceStruct.Spec.Template.Spec.Containers[0].Env, env)
		}
	}
}

func AddLabels(resObject client.Object, grName, subResourceKind string) {
	labels := resObject.GetLabels()
	if len(labels) == 0 {
		labels = map[string]string{}
	}
	labels[consts.LABEL_COMPONENT] = consts.OPERATOR_NAME
	labels[consts.LABEL_DEPLOYED_BY] = fmt.Sprintf("%s.%s", consts.GCURESOURCE, grName)
	labels[consts.LABEL_INSTANCE] = subResourceKind
	resObject.SetLabels(labels)
}

func AddOwnereference(resObject client.Object, gcuResource *v1.GcuResource) {
	resObject.SetOwnerReferences([]metav1.OwnerReference{{
		APIVersion: gcuResource.APIVersion,
		Kind:       gcuResource.Kind,
		Name:       gcuResource.Name,
		UID:        gcuResource.UID,
	}})
}

func CheckDeamonsetRunning(client client.Client, namespacedNames []*k8sTypes.NamespacedName) (bool, error) {
	for _, namespacedName := range namespacedNames {
		daemonset := &appsv1.DaemonSet{}
		err := client.Get(context.TODO(), *namespacedName, daemonset)
		if err != nil {
			logs.Error(err, "get cluster daemonset failed when checking daemonset run status")
			return false, err
		}
		if daemonset.Status.DesiredNumberScheduled <= 0 || daemonset.Status.DesiredNumberScheduled != daemonset.Status.NumberReady {
			logs.Warn("check daemonset(name:%s, namespace:%s) run status is not running", namespacedName.Name,
				namespacedName.Namespace)
			return false, nil
		}
		logs.Info("check daemonset(name:%s, namespace:%s) run status is running", namespacedName.Name,
			namespacedName.Namespace)
	}
	return true, nil
}

func CheckDeploymentsRunning(client client.Client, namespacedNames []*k8sTypes.NamespacedName) (bool, error) {
	for _, namespacedName := range namespacedNames {
		deployment := &appsv1.Deployment{}
		err := client.Get(context.TODO(), *namespacedName, deployment)
		if err != nil {
			logs.Error(err, "get cluster deployment failed when checking deployment run status")
			return false, err
		}
		if deployment.Status.ReadyReplicas != *deployment.Spec.Replicas {
			logs.Warn("check deployment(name:%s, namespace:%s) run status is not running", namespacedName.Name,
				namespacedName.Namespace)
			return false, nil
		}
		logs.Info("check deployment(name:%s, namespace:%s) run status is running", namespacedName.Name,
			namespacedName.Namespace)
	}
	return true, nil
}

func AddReadyResource(readyResource string) string {
	countList := strings.Split(readyResource, "/")
	oldReadyCount := countList[0]
	newReadyCount, _ := strconv.Atoi(oldReadyCount)
	newReadyCount++
	countList[0] = strconv.Itoa(newReadyCount)
	return strings.Join(countList, "/")
}

func DecoratorForCheckController(decoFunc func(client client.Client, namespacedNames []*k8sTypes.NamespacedName) (bool, error),
	client client.Client, namespacedNames []*k8sTypes.NamespacedName) (bool, error) {
	var err error
	for times := 0; times < consts.RetryTimes; times++ {
		isRunning, err := decoFunc(client, namespacedNames)
		if err == nil && isRunning {
			return true, nil
		}
		logs.Warn("In-place retry times:%d", times)
		time.Sleep(time.Second * consts.Interval)
	}
	logs.Warn("In-place retry timeout:%d, will enter the process to retry", consts.Interval*consts.RetryTimes)
	return false, err
}

func DeleteClusterResource(cli client.Client, resInfo v1.ResourceInfo, namespacedName *k8sTypes.NamespacedName) error {
	obj, ok := object.Objects()[resInfo.Kind]
	if !ok {
		msg := fmt.Sprintf("cluster resource kind:%s not registed by gcu operator, can not delete it", resInfo.Kind)
		err := gcuErr.NewError(consts.ACTION_FAIL, msg)
		logs.Error(err)
		return err
	}
	obj.SetName(namespacedName.Name)
	obj.SetNamespace(namespacedName.Namespace)
	err := cli.Delete(context.TODO(), obj, client.PropagationPolicy(metav1.DeletePropagationForeground))
	if err != nil {
		if kubeErr.IsNotFound(err) {
			logs.Info("cluster resource(kind:%s, name:%s, namespace:%s) not found", resInfo.Kind,
				namespacedName.Name, namespacedName.Namespace)
			return nil
		}
		logs.Warn("use operator k8s client delete %s(group: %s, version: %s) failed, error: %s", resInfo.Kind, resInfo.Group,
			resInfo.Version, err.Error())
		if err := deleteClusterResourceUseDyClient(resInfo, namespacedName.Namespace); err != nil {
			return err
		}
	}
	return nil
}

func deleteClusterResourceUseDyClient(resInfo v1.ResourceInfo, namespace string) error {
	dynamicClient, err := kube.GetClientByGVK(resInfo.Group, resInfo.Version, resInfo.Kind, "")
	if err != nil {
		logs.Error(err, "get %s dynamic client failed", resInfo.Kind)
		return gcuErr.NewError(consts.ACTION_RETRY, err.Error())
	}
	deletePolicy := metav1.DeletePropagationForeground
	if err := dynamicClient.Delete(context.TODO(), resInfo.Name, metav1.DeleteOptions{
		PropagationPolicy: &deletePolicy}); err != nil {
		if kubeErr.IsNotFound(err) {
			logs.Info("cluster resource(kind:%s, name:%s, namespace:%s) not found", resInfo.Kind,
				resInfo.Name, namespace)
			return nil
		}
		logs.Error(err, "delete cluster resource(kind:%s, name:%s, namespace:%s) failed", resInfo.Kind,
			resInfo.Name, namespace)
		return gcuErr.NewError(consts.ACTION_RETRY, err.Error())
	}
	logs.Info("use dynamic client delete %s(group: %s, version: %s) success", resInfo.Kind, resInfo.Group,
		resInfo.Version)
	return nil
}

func (ps *ProcessStatus) checkGcuResourceHasDeleted(err error, action string) bool {
	if action == consts.DELETE && err == nil && ps.Procedure.GetProcedureOKStatus() == gcuStatus.CHECK_DELETE_STATUS_FINISH {
		return true
	}
	return false
}

func InstaceResObject(resObj client.Object, baseFields *v1.BaseFields) error {
	if baseFields.Name != "" {
		resObj.SetName(baseFields.Name)
	}
	if baseFields.Namespace != "" {
		resObj.SetNamespace(baseFields.Namespace)
	}
	if baseFields.ImagePullPolicy == "" {
		baseFields.ImagePullPolicy = consts.ImagePullPolicy
	}
	switch realObj := resObj.(type) {
	case *appsv1.DaemonSet:
		instaceDs(realObj, baseFields)
	case *appsv1.Deployment:
		instaceDeploy(realObj, baseFields)
	default:
		return gcuErr.NewError(consts.ACTION_FAIL, fmt.Sprintf("temporarily does not support instantiation of %T type",
			realObj))
	}
	return nil
}

func instaceDs(dsObj *appsv1.DaemonSet, baseFields *v1.BaseFields) {
	// demo: artifact.enflame.cn/galaxy_docker_hub/gcu/gcu-driver:latest
	image := dsObj.Spec.Template.Spec.Containers[0].Image
	tempList := strings.Split(image, "/")
	repo := strings.Join(tempList[:len(tempList)-1], "/")
	nameVersion := strings.Split(tempList[len(tempList)-1], ":")
	imageName, imageVersion := nameVersion[0], nameVersion[1]
	if baseFields.Repository != "" {
		repo = baseFields.Repository
	}
	if baseFields.Image != "" {
		imageName = baseFields.Image
	}
	if baseFields.Version != "" {
		imageVersion = baseFields.Version
	}
	dsObj.Spec.Template.Spec.Containers[0].Image = fmt.Sprintf("%s/%s:%s", repo, imageName, imageVersion)
	if baseFields.ImagePullPolicy != "" {
		dsObj.Spec.Template.Spec.Containers[0].ImagePullPolicy = corev1.PullPolicy(baseFields.ImagePullPolicy)
	}
	AddDsNodeSelector(dsObj, baseFields.NodeSelector)
	AddDsEnv(dsObj, baseFields.Env)
}

func instaceDeploy(deployObj *appsv1.Deployment, baseFields *v1.BaseFields) {
	image := deployObj.Spec.Template.Spec.Containers[0].Image
	tempList := strings.Split(image, "/")
	repo := strings.Join(tempList[:len(tempList)-1], "/")
	nameVersion := strings.Split(tempList[len(tempList)-1], ":")
	imageName, imageVersion := nameVersion[0], nameVersion[1]
	if baseFields.Repository != "" {
		repo = baseFields.Repository
	}
	if baseFields.Image != "" {
		imageName = baseFields.Image
	}
	if baseFields.Version != "" {
		imageVersion = baseFields.Version
	}
	deployObj.Spec.Template.Spec.Containers[0].Image = fmt.Sprintf("%s/%s:%s", repo, imageName, imageVersion)
	if baseFields.ImagePullPolicy != "" {
		deployObj.Spec.Template.Spec.Containers[0].ImagePullPolicy = corev1.PullPolicy(baseFields.ImagePullPolicy)
	}
	AddDeployNodeSelector(deployObj, baseFields.NodeSelector)
	AddDeployEnv(deployObj, baseFields.Env)
}

func AddCommandArgs(resObj client.Object, command, args []string) {
	switch realObj := resObj.(type) {
	case *appsv1.DaemonSet:
		if len(command) > 0 {
			realObj.Spec.Template.Spec.Containers[0].Command = command
		}
		if len(args) > 0 {
			realObj.Spec.Template.Spec.Containers[0].Args = args
		}

	case *appsv1.Deployment:
		if len(command) > 0 {
			realObj.Spec.Template.Spec.Containers[0].Command = command
		}
		if len(args) > 0 {
			realObj.Spec.Template.Spec.Containers[0].Args = args
		}
	default:
		logs.Warn("%s object not support instance container start command", realObj.GetObjectKind().GroupVersionKind().Kind)
	}
}

func GetNamespace(resource, namespace string) string {
	if namespace != "" {
		return namespace
	}
	return consts.DefaultNamespace
}

func GetResourceTypeMeta(resContent string) (*metav1.TypeMeta, error) {
	typeMeta := new(metav1.TypeMeta)
	if err := json.Unmarshal([]byte(resContent), typeMeta); err != nil {
		logs.Error(err, "json unmarshal content: %s to metav1.TypeMeta failed", resContent)
		return nil, err
	}
	return typeMeta, nil
}

func (ps *ProcessStatus) CreateResourceToCluster(resObj client.Object) (err error) {
	gvk := resObj.GetObjectKind().GroupVersionKind()
	defer ps.addResourceInfo(&err, resObj.GetName(), gvk)

	if err = kube.NewResourceClient(ps.Procedure.GetClient(), ps.Procedure.GetGCUResource().Name,
		string(ps.Procedure.GetGCUResource().UID)).Create(resObj); err != nil {
		return gcuErr.NewError(consts.ACTION_RETRY, err.Error())
	}
	return nil
}

func (ps *ProcessStatus) addResourceInfo(err *error, name string, gvk schema.GroupVersionKind) {
	if *err != nil {
		return
	}
	ps.ResourceStatus.OwnResources = append(ps.ResourceStatus.OwnResources, v1.ResourceInfo{
		Group:   gvk.Group,
		Version: gvk.Version,
		Kind:    gvk.Kind,
		Name:    name,
	})
}

func ExistsFileOrDic(path string) (bool, error) {
	_, err := os.Stat(path)
	if err != nil {
		if err.(*os.PathError).Err.Error() != "no such file or directory" {
			logs.Error(err, "os stat file: %s failed", path)
			return false, gcuErr.NewError(consts.ACTION_FAIL, err.Error())
		}
		return false, nil
	}
	return true, nil
}

func VersionGreaterOrEqual(version1, version2 string) (bool, error) {
	v1, err := semver.NewVersion(version1)
	if err != nil {
		logs.Error(err, "invalid version: %s", version1)
		return false, err
	}

	v2, err := semver.NewVersion(version2)
	if err != nil {
		logs.Error(err, "invalid version: %s", version2)
		return false, err
	}
	if v1.GreaterThan(v2) || v1.Equal(v2) {
		return true, nil
	}
	return false, nil
}

func NeedInstallDriver(gcuResource *v1.GcuResource) bool {
	if gcuResource.Spec.Driver == nil || (gcuResource.Spec.Driver.Deploy != nil &&
		!*gcuResource.Spec.Driver.Deploy) {
		return false
	}
	return true
}

func NeedInstallContainerToolkit(gcuResource *v1.GcuResource) bool {
	if gcuResource.Spec.ContainerToolkit == nil || (gcuResource.Spec.ContainerToolkit.Deploy != nil &&
		!*gcuResource.Spec.ContainerToolkit.Deploy) {
		return false
	}
	return true
}
