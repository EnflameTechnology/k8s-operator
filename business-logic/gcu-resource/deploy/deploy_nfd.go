// Copyright (c) 2022 Enflame. All Rights Reserved.

package deploy

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	k8sTypes "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "gcu-operator/apis/topscloud/v1"
	"gcu-operator/business-logic/base/interfaces"
	"gcu-operator/business-logic/gcu-resource/base"
	gcuPro "gcu-operator/business-logic/gcu-resource/base/progress"
	gcuStatus "gcu-operator/business-logic/gcu-resource/base/status"
	"gcu-operator/pkg/common"
	"gcu-operator/pkg/consts"
	gcuErr "gcu-operator/pkg/error"
	"gcu-operator/pkg/logs"
	"gcu-operator/pkg/object"
)

const (
	nfdPodLabelKey   = "app.kubernetes.io/name"
	nfdPodLabelValue = "node-feature-discovery"
)

type DeployNfd struct {
	GcuResource *v1.GcuResource
	Client      client.Client
	Name        string
	UUID        string
}

var _ interfaces.Procedure = &DeployNfd{}

func NewDeployNfd(gcuResource *v1.GcuResource, client client.Client) interfaces.Procedure {
	return &DeployNfd{
		GcuResource: gcuResource,
		Client:      client,
		Name:        gcuResource.Name,
		UUID:        string(gcuResource.UID),
	}
}

func (rer *DeployNfd) GetProcedureName() string {
	return "DeployNfd"
}

func (rer *DeployNfd) GetProcedureStatus() []string {
	return []string{gcuStatus.DEPLOYING, gcuStatus.DEPLOY_NFD_FAILED}
}

func (rer *DeployNfd) GetProcedureOKStatus() string {
	return gcuStatus.DEPLOY_NFD_SUCCESS
}

func (rer *DeployNfd) GetProcedureFailStatus() string {
	return gcuStatus.DEPLOY_NFD_FAILED
}

func (rer *DeployNfd) GetProcedureOKProgress() int {
	return gcuPro.DEPLOY_NFD_SUCCESS
}

func (rer *DeployNfd) GetProcedureFailProgress() int {
	return gcuPro.DEPLOY_NFD_FAILED
}

func (rer *DeployNfd) GetProcedureAction() string {
	return consts.DEPLOY
}

func (rer *DeployNfd) GetGCUResource() *v1.GcuResource {
	return rer.GcuResource
}

func (rer *DeployNfd) GetClient() client.Client {
	return rer.Client
}

func (rer *DeployNfd) Execute() (err error) {
	status := &base.ProcessStatus{Procedure: rer, AddNodesInfo: true}
	defer status.FinallySetStatus(&err)
	var objectSets, deploy = []client.Object{}, true
	defer rer.updateStatus(status, &err, &objectSets, &deploy)

	//logic here
	if rer.GcuResource.Spec.Nfd == nil || (rer.GcuResource.Spec.Nfd.Deploy != nil && !*rer.GcuResource.Spec.Nfd.Deploy) {
		logs.Warn("resource %s need not deploy, will skip this procedure", consts.NFD)
		deploy = false
		return nil
	}
	status.ResourceStatus = &v1.ResourceStatus{
		Name:         consts.NFD,
		Namespace:    base.GetNamespace(consts.NFD, rer.GcuResource.Spec.Nfd.Namespace),
		Status:       gcuStatus.DEPLOYING,
		OwnResources: []v1.ResourceInfo{},
	}
	if err = rer.getObjectSets(&objectSets); err != nil {
		return err
	}
	if err = rer.deployResources(objectSets, status); err != nil {
		return err
	}
	return nil
}

func (rer *DeployNfd) deployResources(objectSets []client.Object, status *base.ProcessStatus) error {
	for _, resObj := range objectSets {
		gvk := resObj.GetObjectKind().GroupVersionKind()
		kind, name, namespace := gvk.Kind, resObj.GetName(), resObj.GetNamespace()
		logs.Info("deploy %s(name: %s, namespace: %s) of %s to cluster start, content: %s", kind, name, namespace,
			consts.NFD, common.ChangeInterfaceToString(resObj))
		if err := status.CreateResourceToCluster(resObj); err != nil {
			return err
		}
		resObj.GetObjectKind().SetGroupVersionKind(gvk)
		logs.Info("deploy %s(name: %s, namespace: %s) of %s to cluster success, gr name:%s, uuid:%s",
			kind, name, namespace, consts.NFD, rer.Name, rer.UUID)
	}
	logs.Info("deploy all resource of %s to cluster success, count: %d", consts.NFD, len(objectSets))
	return nil
}

func (rer *DeployNfd) getObjectSets(objectSets *[]client.Object) error {
	files, err := os.ReadDir(base.NfdPath)
	if err != nil {
		logs.Error(err, "read directory: %s failed", base.NfdPath)
		return gcuErr.NewError(consts.ACTION_FAIL, err.Error())
	}
	for _, file := range files {
		resObj, err := rer.readAndParseBlueprint(file.Name())
		if err != nil {
			return err
		}
		base.AddLabels(resObj, rer.Name, consts.NFD)
		base.AddOwnereference(resObj, rer.GcuResource)
		if rer.GcuResource.Spec.Nfd.Namespace != "" {
			resObj.SetNamespace(rer.GcuResource.Spec.Nfd.Namespace)
		}
		*objectSets = append(*objectSets, resObj)
	}
	logs.Info("read and parse all cluster resource of %s finish", consts.NFD)
	return nil
}

func (rer *DeployNfd) readAndParseBlueprint(fileName string) (client.Object, error) {
	filePath := fmt.Sprintf("%s/%s", base.NfdPath, fileName)
	content, err := base.ReadYamlToJson(filePath)
	if err != nil {
		logs.Error(err, "read file: %s content failed", filePath)
		return nil, err
	}
	resMeta, err := base.GetResourceTypeMeta(content)
	if err != nil {
		return nil, gcuErr.NewError(consts.ACTION_FAIL, err.Error())
	}
	resObj := object.Objects()[resMeta.Kind]
	if err := json.Unmarshal([]byte(content), resObj); err != nil {
		logs.Error(err, "json unmarshal resource: %s to client.Object failed", content)
		return nil, gcuErr.NewError(consts.ACTION_FAIL, err.Error())
	}
	switch realObj := resObj.(type) {
	case *appsv1.DaemonSet:
		rer.instanceNfdWorker(realObj)
		rer.removeGrpcProbesIfUnsupported(&realObj.Spec.Template.Spec)
	case *appsv1.Deployment:
		rer.instanceNfdMaster(realObj)
		rer.removeGrpcProbesIfUnsupported(&realObj.Spec.Template.Spec)
	}
	logs.Info("read %s(name: %s, namespace: %s) of %s success", resObj.GetObjectKind().GroupVersionKind().Kind,
		resObj.GetName(), resObj.GetNamespace(), consts.NFD)
	return resObj, nil
}

func (rer *DeployNfd) instanceNfdWorker(resObj *appsv1.DaemonSet) {
	nfdSpec := rer.GcuResource.Spec.Nfd
	rer.addNodeSelectorForDs(resObj, nfdSpec.Worker.NodeSelector)
	if nfdSpec.Worker.Name != "" {
		resObj.SetName(nfdSpec.Worker.Name)
	}
	if nfdSpec.Worker.Namespace != "" {
		resObj.SetNamespace(nfdSpec.Worker.Namespace)
	}

	rer.instanceNfdWorkerContainer(resObj, nfdSpec.Worker)
	if len(nfdSpec.Worker.Command) > 0 {
		resObj.Spec.Template.Spec.Containers[0].Command = nfdSpec.Worker.Command
	}
	if len(nfdSpec.Worker.Args) > 0 {
		resObj.Spec.Template.Spec.Containers[0].Args = nfdSpec.Worker.Args
	}

	logs.Info("instance ds: %s of %s success", resObj.Name, consts.NFD)
}

func (rer *DeployNfd) addNodeSelectorForDs(resourceStruct *appsv1.DaemonSet, nodeSelector map[string]string) {
	if resourceStruct.Spec.Template.Spec.NodeSelector == nil {
		resourceStruct.Spec.Template.Spec.NodeSelector = nodeSelector
	} else {
		for key, value := range nodeSelector {
			resourceStruct.Spec.Template.Spec.NodeSelector[key] = value
		}
	}
}

func (rer *DeployNfd) addNodeSelectorForDeploy(resourceStruct *appsv1.Deployment, nodeSelector map[string]string) {
	if resourceStruct.Spec.Template.Spec.NodeSelector == nil {
		resourceStruct.Spec.Template.Spec.NodeSelector = nodeSelector
	} else {
		for key, value := range nodeSelector {
			resourceStruct.Spec.Template.Spec.NodeSelector[key] = value
		}
	}
}

func (rer *DeployNfd) instanceNfdWorkerContainer(resObj *appsv1.DaemonSet, baseFields *v1.BaseFields) {
	if baseFields == nil {
		return
	}
	image := resObj.Spec.Template.Spec.Containers[0].Image
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
	resObj.Spec.Template.Spec.Containers[0].Image = fmt.Sprintf("%s/%s:%s", repo, imageName, imageVersion)
	if baseFields.ImagePullPolicy != "" {
		resObj.Spec.Template.Spec.Containers[0].ImagePullPolicy = corev1.PullPolicy(baseFields.ImagePullPolicy)
	}
	resObj.Spec.Template.Spec.Containers[0].Env = append(resObj.Spec.Template.Spec.Containers[0].Env, baseFields.Env...)
}

func (rer *DeployNfd) instanceNfdMaster(resObj *appsv1.Deployment) {
	nfdSpec := rer.GcuResource.Spec.Nfd
	if nfdSpec.Master.Replicas != nil {
		resObj.Spec.Replicas = nfdSpec.Master.Replicas
	}
	rer.addNodeSelectorForDeploy(resObj, nfdSpec.Master.NodeSelector)
	if nfdSpec.Master.Name != "" {
		resObj.SetName(nfdSpec.Master.Name)
	}
	if nfdSpec.Master.Namespace != "" {
		resObj.SetNamespace(nfdSpec.Master.Namespace)
	}

	rer.instanceNfdMasterContainer(resObj, nfdSpec.Master)
	if len(nfdSpec.Master.Command) > 0 {
		resObj.Spec.Template.Spec.Containers[0].Command = nfdSpec.Master.Command
	}
	if len(nfdSpec.Master.Args) > 0 {
		resObj.Spec.Template.Spec.Containers[0].Args = nfdSpec.Worker.Args
	}

	logs.Info("instance deployment: %s of %s success", resObj.Name, consts.NFD)
}

func (rer *DeployNfd) instanceNfdMasterContainer(resObj *appsv1.Deployment, baseFields *v1.BaseFields) {
	if baseFields == nil {
		return
	}
	image := resObj.Spec.Template.Spec.Containers[0].Image
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
	resObj.Spec.Template.Spec.Containers[0].Image = fmt.Sprintf("%s/%s:%s", repo, imageName, imageVersion)
	if baseFields.ImagePullPolicy != "" {
		resObj.Spec.Template.Spec.Containers[0].ImagePullPolicy = corev1.PullPolicy(baseFields.ImagePullPolicy)
	}
	resObj.Spec.Template.Spec.Containers[0].Env = append(resObj.Spec.Template.Spec.Containers[0].Env, baseFields.Env...)
	rer.instanceNfdMasterInitContainer(resObj, baseFields)
}

func (rer *DeployNfd) instanceNfdMasterInitContainer(resObj *appsv1.Deployment, baseFields *v1.BaseFields) {
	if baseFields == nil {
		return
	}
	image := resObj.Spec.Template.Spec.InitContainers[0].Image
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
	resObj.Spec.Template.Spec.InitContainers[0].Image = fmt.Sprintf("%s/%s:%s", repo, imageName, imageVersion)
	if baseFields.ImagePullPolicy != "" {
		resObj.Spec.Template.Spec.InitContainers[0].ImagePullPolicy = corev1.PullPolicy(baseFields.ImagePullPolicy)
	}
}

// removeGrpcProbesIfUnsupported strips the gRPC startup/liveness/readiness
// probes from the NFD workloads when the target cluster is older than v1.24.
// gRPC probes are only supported since Kubernetes 1.24 (beta, enabled by
// default); older API servers reject the grpc probe field and the workload
// fails to apply.
func (rer *DeployNfd) removeGrpcProbesIfUnsupported(podSpec *corev1.PodSpec) {
	kubeletVersion := ""
	for nodeName, nodeInfo := range rer.GcuResource.Status.NodesInfo {
		if nodeInfo.IsGCUNode {
			kubeletVersion = nodeInfo.KubeletVersion
			logs.Info("detected node: %s kubelet version: %s", nodeName, kubeletVersion)
			break
		}
	}
	versionGe, err := base.VersionGreaterOrEqual(kubeletVersion, "v1.24.0")
	if err != nil {
		logs.Error(err, "parse kubelet version: %s failed, keep gRPC probes of %s", kubeletVersion, consts.NFD)
		return
	}
	if versionGe {
		return
	}
	logs.Warn("kubernetes version < 1.24: gRPC probes are not supported, removing probes from %s workloads", consts.NFD)
	for i := range podSpec.Containers {
		podSpec.Containers[i].StartupProbe = nil
		podSpec.Containers[i].LivenessProbe = nil
		podSpec.Containers[i].ReadinessProbe = nil
	}
}

func (rer *DeployNfd) waitNodeLabeled() error {
	nfdPodList := &corev1.PodList{}
	labelSelector := map[string]string{nfdPodLabelKey: nfdPodLabelValue, "role": "worker"}
	listOption := []client.ListOption{client.MatchingLabelsSelector{Selector: labels.SelectorFromSet(labelSelector)}}
	if err := rer.Client.List(context.TODO(), nfdPodList, listOption...); err != nil {
		logs.Error(err, "list nfd pod list failed when waitNodeLabeled")
		return gcuErr.NewError(consts.ACTION_RETRY, err.Error())
	}
	logs.Info("There are %d nodes with nfd installed", len(nfdPodList.Items))

	for {
		allLabeled := true
		for _, pod := range nfdPodList.Items {
			node := &corev1.Node{}
			nodeName := pod.Spec.NodeName
			if err := rer.Client.Get(context.TODO(), k8sTypes.NamespacedName{Name: nodeName}, node); err != nil {
				logs.Error(err, "get cluster node: %s detail failed", nodeName)
				return err
			}
			if !rer.nodeHasLabeledByNfd(node.GetLabels()) {
				logs.Warn("check node %s not labeled by nfd, waiting...", nodeName)
				allLabeled = false
				time.Sleep(time.Second * 1)
				break
			}
			logs.Info("check node %s has labeled by nfd", nodeName)
		}

		if allLabeled {
			logs.Info("check all nfd node labeled success")
			return nil
		}
	}
}

func (rer *DeployNfd) nodeHasLabeledByNfd(labels map[string]string) bool {
	for key := range labels {
		if strings.HasPrefix(key, consts.NfdLabelPrefix) {
			return true
		}
	}
	return false
}

// updateStatus only used for modify readyResource
func (rer *DeployNfd) updateStatus(status *base.ProcessStatus, err *error, objectSets *[]client.Object,
	deploy *bool) {
	if *err != nil || !*deploy {
		return
	}
	// check nfd ds run status
	namespacedNames := []*k8sTypes.NamespacedName{}
	for _, resObj := range *objectSets {
		if resObj.GetObjectKind().GroupVersionKind().Kind == object.DaemonSet {
			namespacedNames = append(namespacedNames, &k8sTypes.NamespacedName{
				Namespace: resObj.GetNamespace(),
				Name:      resObj.GetName(),
			})
		}
	}
	isRunning, errCheck := base.DecoratorForCheckController(base.CheckDeamonsetRunning, rer.Client, namespacedNames)
	// check daemonset status failed, return and requeue reconciler
	if errCheck != nil {
		*err = errCheck
		return
	}
	// check daemonset running, nfd deploy ok
	if isRunning {
		status.IsReady = true
		logs.Info("check all resource of %s run status running, wait all nodes labeled", consts.NFD)
		if errWait := rer.waitNodeLabeled(); err != nil {
			*err = errWait
		}
	} else {
		msg := fmt.Sprintf("check ds of %s of gr name:%s, uuid:%s is not running", consts.NFD, rer.Name, rer.UUID)
		logs.Warn(msg)
		*err = gcuErr.NewError(consts.ACTION_RETRY, msg)
	}
}
