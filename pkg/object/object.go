// Copyright (c) 2022 Enflame. All Rights Reserved.

package object

import (
	admissionv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	nodev1 "k8s.io/api/node/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	schedv1 "k8s.io/api/scheduling/v1beta1"
	extensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	Namespace                      = "Namespace"
	Deployment                     = "Deployment"
	DaemonSet                      = "DaemonSet"
	StatefulSet                    = "StatefulSet"
	ReplicaSet                     = "ReplicaSet"
	ReplicationController          = "ReplicationController"
	ServiceAccount                 = "ServiceAccount"
	ConfigMap                      = "ConfigMap"
	Pod                            = "Pod"
	Service                        = "Service"
	Node                           = "Node"
	Role                           = "Role"
	RoleBinding                    = "RoleBinding"
	ClusterRole                    = "ClusterRole"
	ClusterRoleBinding             = "ClusterRoleBinding"
	PriorityClass                  = "PriorityClass"
	RuntimeClass                   = "RuntimeClass"
	PersistentVolume               = "PersistentVolume"
	PersistentVolumeClaim          = "PersistentVolumeClaim"
	CustomResourceDefinition       = "CustomResourceDefinition"
	Job                            = "Job"
	MutatingWebhookConfiguration   = "MutatingWebhookConfiguration"
	ValidatingWebhookConfiguration = "ValidatingWebhookConfiguration"
)

func Objects() map[string]client.Object {
	return map[string]client.Object{
		Deployment:                     &appsv1.Deployment{},
		DaemonSet:                      &appsv1.DaemonSet{},
		StatefulSet:                    &appsv1.StatefulSet{},
		ReplicaSet:                     &appsv1.ReplicaSet{},
		Namespace:                      &corev1.Namespace{},
		ReplicationController:          &corev1.ReplicationController{},
		ServiceAccount:                 &corev1.ServiceAccount{},
		ConfigMap:                      &corev1.ConfigMap{},
		Pod:                            &corev1.Pod{},
		PersistentVolume:               &corev1.PersistentVolume{},
		PersistentVolumeClaim:          &corev1.PersistentVolumeClaim{},
		Service:                        &corev1.Service{},
		Node:                           &corev1.Node{},
		Role:                           &rbacv1.Role{},
		RoleBinding:                    &rbacv1.RoleBinding{},
		ClusterRole:                    &rbacv1.ClusterRole{},
		ClusterRoleBinding:             &rbacv1.ClusterRoleBinding{},
		PriorityClass:                  &schedv1.PriorityClass{},
		RuntimeClass:                   &nodev1.RuntimeClass{},
		CustomResourceDefinition:       &extensionsv1.CustomResourceDefinition{},
		Job:                            &batchv1.Job{},
		MutatingWebhookConfiguration:   &admissionv1.MutatingWebhookConfiguration{},
		ValidatingWebhookConfiguration: &admissionv1.ValidatingWebhookConfiguration{},
	}
}

var ObjectsList = map[string]client.ObjectList{
	Deployment:                     &appsv1.DeploymentList{},
	DaemonSet:                      &appsv1.DaemonSetList{},
	StatefulSet:                    &appsv1.StatefulSetList{},
	ReplicaSet:                     &appsv1.ReplicaSetList{},
	Namespace:                      &corev1.NamespaceList{},
	ReplicationController:          &corev1.ReplicationControllerList{},
	ServiceAccount:                 &corev1.ServiceAccountList{},
	ConfigMap:                      &corev1.ConfigMapList{},
	Pod:                            &corev1.PodList{},
	PersistentVolume:               &corev1.PersistentVolumeList{},
	PersistentVolumeClaim:          &corev1.PersistentVolumeClaimList{},
	Service:                        &corev1.ServiceList{},
	Node:                           &corev1.NodeList{},
	Role:                           &rbacv1.RoleList{},
	RoleBinding:                    &rbacv1.RoleBindingList{},
	ClusterRole:                    &rbacv1.ClusterRoleList{},
	ClusterRoleBinding:             &rbacv1.ClusterRoleBindingList{},
	PriorityClass:                  &schedv1.PriorityClassList{},
	RuntimeClass:                   &nodev1.RuntimeClassList{},
	CustomResourceDefinition:       &extensionsv1.CustomResourceDefinitionList{},
	Job:                            &batchv1.JobList{},
	MutatingWebhookConfiguration:   &admissionv1.MutatingWebhookConfigurationList{},
	ValidatingWebhookConfiguration: &admissionv1.ValidatingWebhookConfigurationList{},
}
