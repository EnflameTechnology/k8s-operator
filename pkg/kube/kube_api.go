// Copyright (c) 2022 Enflame. All Rights Reserved.

package kube

import (
	"os"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"gcu-operator/pkg/logs"
)

const RecommendedKubeConfigPathEnv = "KUBECONFIG"

// GetK8sConfig for get k8s config
func GetK8sConfig() (*rest.Config, error) {
	kubeConfig := ""
	if len(os.Getenv(RecommendedKubeConfigPathEnv)) > 0 {
		// use the current context in kubeconfig
		// This is very useful for running locally.
		kubeConfig = os.Getenv(RecommendedKubeConfigPathEnv)
	}

	// kubeconfig := "/.kube/config"
	config, err := clientcmd.BuildConfigFromFlags("", kubeConfig)
	if err != nil {
		logs.Error(err, "get k8s config failed")
		return nil, err
	}
	return config, err
}

func GetResourceMapper(group, version, kind string, config *rest.Config) (*meta.RESTMapping, error) {
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		logs.Error(err, "get discovery client failed")
		return nil, err
	}
	mapperGVRGVK := restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(discoveryClient))
	gvk := &schema.GroupVersionKind{
		Group:   group,
		Version: version,
		Kind:    kind,
	}
	resourceMapper, err := mapperGVRGVK.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		logs.Error(err, "get resource mapper failed")
		return nil, err
	}
	return resourceMapper, nil
}

func IsNamespaced(resourceMapper *meta.RESTMapping) bool {
	return resourceMapper.Scope.Name() == meta.RESTScopeNameNamespace
}

// GetClient for get dynamic client
func GetClientByGVK(group, version, kind, namespace string) (dynamic.ResourceInterface, error) {
	config, err := GetK8sConfig()
	if err != nil {
		return nil, err
	}

	resourceMapper, err := GetResourceMapper(group, version, kind, config)
	if err != nil {
		return nil, err
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		logs.Error(err, "get dynamic client failed")
		return nil, err
	}

	if IsNamespaced(resourceMapper) {
		return dynamicClient.Resource(resourceMapper.Resource).Namespace(namespace), nil
	}
	return dynamicClient.Resource(resourceMapper.Resource), nil
}

func GetClientByObject(obj client.Object, namespace string) (dynamic.ResourceInterface, error) {
	config, err := GetK8sConfig()
	if err != nil {
		return nil, err
	}
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		logs.Error(err, "get discovery client failed")
		return nil, err
	}
	mapperGVRGVK := restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(discoveryClient))
	gvk := obj.GetObjectKind().GroupVersionKind()
	resourceMapper, err := mapperGVRGVK.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		logs.Error(err, "get resource mapper failed")
		return nil, err
	}
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		logs.Error(err, "get dynamic client failed")
		return nil, err
	}

	if IsNamespaced(resourceMapper) {
		return dynamicClient.Resource(resourceMapper.Resource).Namespace(namespace), nil
	}
	return dynamicClient.Resource(resourceMapper.Resource), nil
}

func GetClientSet() (*kubernetes.Clientset, error) {
	kubeConfig := ""
	if len(os.Getenv(RecommendedKubeConfigPathEnv)) > 0 {
		// use the current context in kubeconfig
		// This is very useful for running locally.
		kubeConfig = os.Getenv(RecommendedKubeConfigPathEnv)
	}

	// Get kubernetes config.
	restConfig, err := clientcmd.BuildConfigFromFlags("", kubeConfig)
	if err != nil {
		logs.Error(err, "build rest config failed")
		return nil, err
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		logs.Error(err, "get kube client set failed")
		return nil, err
	}
	logs.Info("get kube clientSet success")
	return clientset, nil
}
