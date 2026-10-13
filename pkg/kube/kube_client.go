// Copyright (c) 2022 Enflame. All Rights Reserved.

package kube

import (
	"context"
	"encoding/json"
	"fmt"

	kubeErr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	k8sTypes "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"gcu-operator/pkg/logs"
)

type ResourceClient struct {
	opClient client.Client
	grName   string
	grUUID   string
}

func NewResourceClient(opClient client.Client, grName, grUUID string) *ResourceClient {
	return &ResourceClient{
		opClient: opClient,
		grName:   grName,
		grUUID:   grUUID,
	}
}

func (rer *ResourceClient) Create(resObj client.Object) error {
	gvk := resObj.GetObjectKind().GroupVersionKind()
	name, namespace := resObj.GetName(), resObj.GetNamespace()
	err := rer.Get(resObj, k8sTypes.NamespacedName{Namespace: namespace, Name: name})
	if err != nil {
		if !kubeErr.IsNotFound(err) {
			return err
		}
		if err := rer.opClient.Create(context.TODO(), resObj); err != nil {
			if err := rer.CreateResourceUseDyClient(resObj); err != nil {
				logs.Error(err, "create %s(name: %s, namespace: %s, group: %s, version: %s) to cluster failed", gvk.Kind,
					resObj.GetName(), resObj.GetNamespace(), gvk.Group, gvk.Version)
				return err
			}
		}
		return nil
	}
	ownerReferences := resObj.GetOwnerReferences()
	for _, owner := range ownerReferences {
		if owner.Name == rer.grName && owner.UID == k8sTypes.UID(rer.grUUID) {
			logs.Info("create %s(name: %s, namespace: %s, group: %s, version: %s) to cluster already exist", gvk.Kind,
				resObj.GetName(), resObj.GetNamespace(), gvk.Group, gvk.Version)
			return nil
		}
	}
	msg := fmt.Sprintf("create %s(name: %s, namespace: %s, group: %s, version: %s) to cluster conflict", gvk.Kind,
		resObj.GetName(), resObj.GetNamespace(), gvk.Group, gvk.Version)
	logs.Error(fmt.Errorf(msg))
	return fmt.Errorf(msg)
}

func (rer *ResourceClient) Get(resObj client.Object, namespacedName k8sTypes.NamespacedName) error {
	err := rer.opClient.Get(context.TODO(), namespacedName, resObj)
	if err == nil {
		return nil
	}
	if kubeErr.IsNotFound(err) {
		return err
	}
	gvk := resObj.GetObjectKind().GroupVersionKind()
	dynamicClient, err := GetClientByGVK(gvk.Group, gvk.Version, gvk.Kind, "")
	if err != nil {
		logs.Error(err, "get %s dynamic client failed", gvk.Kind)
		return err
	}
	unstructuredObj, err := dynamicClient.Get(context.TODO(), resObj.GetName(), metav1.GetOptions{})
	if err != nil {
		if kubeErr.IsNotFound(err) {
			return err
		}
		logs.Error(err, "get %s(name: %s, namespace: %s, group: %s, version: %s) failed from cluster", gvk.Kind,
			resObj.GetName(), resObj.GetNamespace(), gvk.Group, gvk.Version)
		return err
	}
	resByte, err := unstructuredObj.MarshalJSON()
	if err != nil {
		logs.Error(err, "json marshal unstructured to bytes failed, content: %v", unstructuredObj.Object)
		return err
	}
	if err := json.Unmarshal(resByte, resObj); err != nil {
		logs.Error(err, "json unmarshal bytes to client.Object failed, content: %s", string(resByte))
		return err
	}
	return nil
}

func (rer *ResourceClient) Delete() {}

func (rer *ResourceClient) CreateResourceUseDyClient(resObj client.Object) error {
	gvk := resObj.GetObjectKind().GroupVersionKind()
	dynamicClient, err := GetClientByGVK(gvk.Group, gvk.Version, gvk.Kind, "")
	if err != nil {
		logs.Error(err, "get %s dynamic client failed", gvk.Kind)
		return err
	}
	resByte, err := json.Marshal(resObj)
	if err != nil {
		logs.Error(err, "json marshal %s to bytes failed, resObj: %v", gvk.Kind, resObj)
		return err
	}
	mapObj := new(map[string]interface{})
	if err := json.Unmarshal(resByte, mapObj); err != nil {
		logs.Error(err, "json unmarshal %s to map failed, resource: %s", gvk.Kind, string(resByte))
		return err
	}
	unstructuredObj := &unstructured.Unstructured{Object: *mapObj}
	_, err = dynamicClient.Create(context.TODO(), unstructuredObj, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	return nil
}
