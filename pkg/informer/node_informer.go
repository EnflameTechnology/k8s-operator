// Copyright (c) 2022 Enflame. All Rights Reserved.

package informer

import (
	"context"
	"encoding/json"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	corev1 "k8s.io/api/core/v1"
	kubeErr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	listerv1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "gcu-operator/apis/topscloud/v1"
	"gcu-operator/pkg/consts"
	"gcu-operator/pkg/kube"
	"gcu-operator/pkg/logs"
)

const RecommendedKubeConfigPathEnv = "KUBECONFIG"

type NodeController struct {
	nodeInformer cache.SharedIndexInformer
	lister       listerv1.NodeLister
	queue        workqueue.RateLimitingInterface
	opClient     client.Client
	clientSet    *kubernetes.Clientset
}

func NewNodeController(opClient client.Client) (*NodeController, error) {
	clientSet, err := kube.GetClientSet()
	if err != nil {
		return nil, err
	}
	factory := informers.NewSharedInformerFactory(clientSet, time.Minute*10)
	nodeInformer := factory.Core().V1().Nodes().Informer()
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
	nodeController := &NodeController{
		nodeInformer: nodeInformer,
		lister:       factory.Core().V1().Nodes().Lister(),
		queue:        queue,
		opClient:     opClient,
		clientSet:    clientSet,
	}

	nodeController.nodeInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			logs.Info("node: %s was added to node informer, and put in controller queue", obj.(*corev1.Node).Name)
			nodeController.addQueue(obj)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			newNode := newObj.(*corev1.Node)
			if ok, action := nodeController.needUpdateUnifiedDeivceLabel(newNode); ok {
				logs.Info("node: %s need %s unified device label: %s=%s", newNode.Name, action,
					consts.UnifiedDeviceLabelKey, consts.UnifiedDeviceLabelValue)
				nodeController.addQueue(newObj)
			}
		},
		DeleteFunc: func(obj interface{}) {
			logs.Info("node: %s was deleted from node informer, and put in controller queue", obj.(*corev1.Node).Name)
			nodeController.addQueue(obj)
		},
	})
	return nodeController, nil
}

func (rer *NodeController) needUpdateUnifiedDeivceLabel(node *corev1.Node) (bool, string) {
	labelSelectorForEnflameVendorID := labels.SelectorFromSet(consts.EnflameVendorIdLabelsMap)
	labelSelectorForUnifiedDeivceLabel := labels.SelectorFromSet(labels.Set{
		consts.UnifiedDeviceLabelKey: consts.UnifiedDeviceLabelValue,
	})
	if labelSelectorForEnflameVendorID.Matches(labels.Set(node.Labels)) {
		if !labelSelectorForUnifiedDeivceLabel.Matches(labels.Set(node.Labels)) {
			return true, "add"
		}
	} else if labelSelectorForUnifiedDeivceLabel.Matches(labels.Set(node.Labels)) {
		return true, "remove"
	}
	return false, ""
}

func (rer *NodeController) Run() {
	defer runtime.HandleCrash()
	defer rer.queue.ShutDown()

	logs.Info("Starting node informer")
	stopCh := make(chan struct{})
	go func() {
		stopSigs := make(chan os.Signal, 2)
		signal.Notify(stopSigs, []os.Signal{os.Interrupt, syscall.SIGTERM}...)
		<-stopSigs
		logs.Info("Stop node informer")
		close(stopCh)
	}()
	go rer.nodeInformer.Run(stopCh)

	if !cache.WaitForCacheSync(stopCh, rer.nodeInformer.HasSynced) {
		logs.Error("failed to sync caches")
		return
	}
	go rer.runWorker()
	<-stopCh
}

func (rer *NodeController) runWorker() {
	for rer.processNextItem() {
	}
}

func (rer *NodeController) processNextItem() bool {
	key, quit := rer.queue.Get()
	if quit {
		logs.Info("controller queue stopped")
		return false
	}
	defer rer.queue.Done(key)

	err := rer.syncHandler(key.(string))
	if err != nil {
		logs.Warn("controller queue syncHandler failed, waiting for retry...")
		rer.queue.AddRateLimited(key)
	} else {
		rer.queue.Forget(key)
	}
	return true
}

func (rer *NodeController) syncHandler(key string) error {
	node, err := rer.lister.Get(key)
	if err != nil {
		if kubeErr.IsNotFound(err) {
			logs.Info("cluster node: %s has been deleted, will remove it from gcuresources", key)
			return rer.removeGCUNodeByName(key)
		}
		logs.Error(err, "get node: %s from cluster failed")
		return err
	}
	if err = rer.updateUnifiedDeivceLabel(node); err != nil {
		return err
	}
	return rer.updateGCUNode(node)
}

func (rer *NodeController) updateUnifiedDeivceLabel(node *corev1.Node) error {
	ok, action := rer.needUpdateUnifiedDeivceLabel(node)
	if !ok {
		logs.Info("node: %s need not to update unified device label, skip it", node.Name)
		return nil
	}
	if action == "add" {
		return rer.setUnifiedDeviceLabel(node)
	}
	return rer.removeUnifiedDeviceLabel(node)
}

func (rer *NodeController) addQueue(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		logs.Error(err)
		return
	}
	rer.queue.Add(key)
}

func (rer *NodeController) removeUnifiedDeviceLabel(node *corev1.Node) error {
	patchData := []map[string]string{
		{
			"op":   "remove",
			"path": "/metadata/labels/" + escapeJSONPointer(consts.UnifiedDeviceLabelKey),
		},
	}
	patchBytes, err := json.Marshal(patchData)
	if err != nil {
		logs.Error(err, "json marshal failed, content: %v", patchData)
		return err
	}
	newNode, err := rer.clientSet.CoreV1().Nodes().Patch(context.TODO(), node.Name, k8stypes.JSONPatchType, patchBytes,
		metav1.PatchOptions{})
	if err != nil {
		logs.Error(err, "patch node: %s remove unified device label failed", node.Name)
		return err
	}
	*node = *newNode
	logs.Info("patch node: %s remove unified device label success", node.Name)
	return nil
}

func (rer *NodeController) setUnifiedDeviceLabel(node *corev1.Node) error {
	patchData := map[string]interface{}{
		"metadata": map[string]interface{}{
			"labels": map[string]string{
				consts.UnifiedDeviceLabelKey: consts.UnifiedDeviceLabelValue,
			},
		},
	}
	patchBytes, err := json.Marshal(patchData)
	if err != nil {
		logs.Error(err, "json marshal failed, content: %v", patchData)
		return err
	}
	newNode, err := rer.clientSet.CoreV1().Nodes().Patch(context.TODO(), node.Name, k8stypes.MergePatchType, patchBytes,
		metav1.PatchOptions{})
	if err != nil {
		logs.Error(err, "patch node: %s set unified device label failed", node.Name)
		return err
	}
	*node = *newNode
	logs.Info("patch node: %s set unified device label success", node.Name)
	return nil
}

func (rer *NodeController) updateGCUNode(node *corev1.Node) error {
	gcuResourceList := &v1.GcuResourceList{}
	if err := rer.opClient.List(context.TODO(), gcuResourceList, &client.ListOptions{}); err != nil {
		logs.Error(err, "list gcuresource failed")
		return err
	}
	if len(gcuResourceList.Items) == 0 {
		return nil
	}
	for i := range gcuResourceList.Items {
		needUpdate := false
		_, isGCUNode := node.Labels[consts.UnifiedDeviceLabelKey]
		gcuResource := &gcuResourceList.Items[i]
		if _, ok := gcuResource.Status.NodesInfo[node.Name]; !ok {
			needUpdate = true
		} else if gcuResource.Status.NodesInfo[node.Name].IsGCUNode != isGCUNode {
			needUpdate = true
		}
		if needUpdate {
			if gcuResource.Status.NodesInfo == nil {
				gcuResource.Status.NodesInfo = map[string]v1.NodeInfo{}
			}
			patch := client.MergeFrom(gcuResource.DeepCopy())
			gcuResource.Status.NodesInfo[node.Name] = v1.NodeInfo{
				IsGCUNode:               isGCUNode,
				KubeletVersion:          node.Status.NodeInfo.KubeletVersion,
				ContainerRuntimeVersion: node.Status.NodeInfo.ContainerRuntimeVersion,
			}
			if err := rer.opClient.Status().Patch(context.TODO(), gcuResource, patch); err != nil {
				logs.Error(err, "patch gcuResource(name:%s, uuid:%s) status node: %s information failed, content: %v",
					gcuResource.Name, gcuResource.UID, node.Name, gcuResource.Status.NodesInfo[node.Name])
				return err
			}
			logs.Info("patch gcuResource(name:%s, uuid:%s) status node: %s information success, content: %v",
				gcuResource.Name, gcuResource.UID, node.Name, gcuResource.Status.NodesInfo[node.Name])
		}
	}
	return nil
}

func (rer *NodeController) removeGCUNodeByName(nodeName string) error {
	gcuResourceList := &v1.GcuResourceList{}
	if err := rer.opClient.List(context.TODO(), gcuResourceList, &client.ListOptions{}); err != nil {
		logs.Error(err, "list gcuresource failed")
		return err
	}
	if len(gcuResourceList.Items) == 0 {
		return nil
	}
	for i := range gcuResourceList.Items {
		gcuResource := &gcuResourceList.Items[i]
		if _, ok := gcuResource.Status.NodesInfo[nodeName]; !ok {
			continue
		}
		patch := client.MergeFrom(gcuResource.DeepCopy())
		delete(gcuResource.Status.NodesInfo, nodeName)
		if err := rer.opClient.Status().Patch(context.TODO(), gcuResource, patch); err != nil {
			logs.Error(err, "patch gcuResource(name: %s, uuid: %s) status remove gcu node: %s failed",
				gcuResource.Name, gcuResource.UID, nodeName)
			return err
		}
		logs.Info("patch gcuResource(name: %s, uuid: %s) status remove gcu node: %s success",
			gcuResource.Name, gcuResource.UID, nodeName)
	}
	return nil
}

func escapeJSONPointer(key string) string {
	key = strings.ReplaceAll(key, "~", "~0")
	key = strings.ReplaceAll(key, "/", "~1")
	return key
}
