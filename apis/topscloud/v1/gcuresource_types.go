// Copyright (c) 2022 Enflame. All Rights Reserved.

package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

type BaseFields struct {
	// Name define the name of component
	// +optional
	Name string `json:"name,omitempty"`

	// Namespace define the namespace of component controller
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// Image define the image name of component
	// +optional
	Image string `json:"image,omitempty"`

	// Version define the image version of component
	// +optional
	Version string `json:"version,omitempty"`

	// Repository define the image repo of component
	// +optional
	Repository string `json:"repository,omitempty"`

	// ImagePullPolicy define the imagePullPolicy of component
	// +optional
	ImagePullPolicy string `json:"imagePullPolicy,omitempty"`

	// NodeSelector define the component can only be deployed on nodes that contain specific labels
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// Env define the env for component
	// +optional
	Env []corev1.EnvVar `json:"env,omitempty"`

	// Command define container start command
	// +optional
	Command []string `json:"command,omitempty"`

	// Args define container start args
	// +optional
	Args []string `json:"args,omitempty"`

	// Replicas define the replicas of deployment controller
	// +optional
	Replicas *int32 `json:"replicas,omitempty"`
}

type DriverSpec struct {
	BaseFields `json:",omitempty"`
	// Deploy define the component need deploy or not
	// +optional
	Deploy *bool `json:"deploy,omitempty"`
	// OneShot define whether to install enflame driver in one shot mode
	// +optional
	OneShot *bool `json:"oneShot,omitempty"`
	// NoS60GCheck define whether to check s60g
	// +optional
	NoS60GCheck *bool `json:"noS60GCheck,omitempty"`
}

type ContainerToolkitSpec struct {
	BaseFields `json:",omitempty"`
	// Deploy define the component need deploy or not
	// +optional
	Deploy *bool `json:"deploy,omitempty"`
}

type DevicePluginSpec struct {
	BaseFields `json:",omitempty"`
	// Deploy define the component need deploy or not
	// +optional
	Deploy *bool `json:"deploy,omitempty"`
	// Variant define the variant of k8s device plugin
	// +optional
	Variant string `json:"variant,omitempty"`
}

type GcuExporterSpec struct {
	BaseFields `json:",omitempty"`
	// Deploy define the component need deploy or not
	// +optional
	Deploy *bool `json:"deploy,omitempty"`
}

type NodeExporterSpec struct {
	BaseFields `json:",omitempty"`
	// Deploy define the component need deploy or not
	// +optional
	Deploy *bool `json:"deploy,omitempty"`
}

type GfdSpec struct {
	BaseFields `json:",omitempty"`
	// Deploy define the component need deploy or not
	// +optional
	Deploy *bool `json:"deploy,omitempty"`
}

type K8sDriverManagerSpec struct {
	BaseFields `json:",omitempty"`
	// Deploy define the component need deploy or not
	// +optional
	Deploy *bool `json:"deploy,omitempty"`
}

type GCUShareSpec struct {
	// Deploy define the component need deploy or not
	// +optional
	Deploy *bool `json:"deploy,omitempty"`
	// DevicePlugin define the information of gcushare device plugin
	// +optional
	DevicePlugin GCUShareDevicePluginSpec `json:"devicePlugin,omitempty"`
	// SchedulerPlugin define the information of gcushare scheduler plugin
	// +optional
	SchedulerPlugin GCUShareSchedulerPluginSpec `json:"schedulerPlugin,omitempty"`
}

type GCUShareDevicePluginSpec struct {
	BaseFields `json:",omitempty"`
	// SliceCount define the count a GCU is shared into
	// +optional
	SliceCount *int `json:"sliceCount,omitempty"`
	// gcushareNodes define the nodes name list you want to share GCU, default all
	// +optional
	GCUshareNodes []string `json:"gcushareNodes,omitempty"`
}

type GCUShareSchedulerPluginSpec struct {
	BaseFields `json:",omitempty"`
}

type NfdSpec struct {
	// Deploy define the component need deploy or not
	// +optional
	Deploy *bool `json:"deploy,omitempty"`
	// Namespace define the namespace of nfd
	// +optional
	Namespace string `json:"namespace,omitempty"`
	// Master define the nfd master detail of nfd
	// +optional
	Master *BaseFields `json:"master,omitempty"`
	// Worker define the nfd worker detail of nfd
	// +optional
	Worker *BaseFields `json:"worker,omitempty"`
}

// GcuResourceSpec defines the desired state of GcuResource
type GcuResourceSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Driver define the enflame driver information
	// +optional
	Nfd *NfdSpec `json:"nfd,omitempty"`

	// Driver define the enflame driver information
	// +optional
	Driver *DriverSpec `json:"driver,omitempty"`

	// ContainerToolkit define the enflame container toolkit information
	// +optional
	ContainerToolkit *ContainerToolkitSpec `json:"containerToolkit,omitempty"`

	// DevicePlugin define the enflame k8s device plugin information
	// +optional
	DevicePlugin *DevicePluginSpec `json:"devicePlugin,omitempty"`

	// GcuExporter define the enflame gcu exporter information
	// +optional
	GcuExporter *GcuExporterSpec `json:"gcuExporter,omitempty"`

	// NodeExporter define the enflame node exporter information
	// +optional
	NodeExporter *NodeExporterSpec `json:"nodeExporter,omitempty"`

	// Gfd define the enflame gcu feature discovery information
	// +optional
	Gfd *GfdSpec `json:"gfd,omitempty"`

	// GCUShare define the gcushare information
	// +optional
	GCUShare *GCUShareSpec `json:"gcushare,omitempty"`

	// K8sDriverManager define the k8sDriverManager information
	// +optional
	K8sDriverManager *K8sDriverManagerSpec `json:"k8sDriverManager,omitempty"`
}

// ResourceStatuses define the status of sub resource
type ResourceStatus struct {
	// Important: Run "make" to regenerate code after modifying this file
	// Name define the sub resource name
	Name string `json:"name,omitempty"`

	// Namespace define the sub resource namespace
	Namespace string `json:"namespace,omitempty"`

	// Status define the sub resource namespace status
	// +optional
	Status string `json:"status,omitempty"`

	// OwnResources define cluster resources owned by this component
	OwnResources []ResourceInfo `json:"ownResources,omitempty"`
}

type ResourceInfo struct {
	// Group define cluster resources group
	Group string `json:"group,omitempty"`
	// Version define cluster resources version
	Version string `json:"version,omitempty"`
	// Kind define cluster resources kind
	Kind string `json:"kind,omitempty"`
	// Name define cluster resources name
	Name string `json:"name,omitempty"`
}

// Conditions define the condtion of gcu-resource
type Condition struct {
	// Important: Run "make" to regenerate code after modifying this file
	// Action define the current action of gcu resource
	Action string `json:"action,omitempty"`

	// Status define the current action status of gcu resource
	Status string `json:"status,omitempty"`

	// Reason define the current action reason of gcu resource
	// +optional
	Reason string `json:"reason,omitempty"`

	// Message define the current action message of gcu resource
	// +optional
	Message string `json:"message,omitempty"`

	// Progress define the current progress of gcu resource
	Progress int `json:"progress,omitempty"`

	// LastUpdateTime define the current action lastUpdateTime of gcu resource
	LastUpdateTime *metav1.Time `json:"lastUpdateTime,omitempty"`
}

// GcuResourceStatus defines the observed state of GcuResource
type GcuResourceStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Phase define the gcu resource run status
	Phase string `json:"phase,omitempty"`
	// ReadyResource define the gcu resource count of is running
	// +optional
	ReadyResource string `json:"readyResource,omitempty"`
	// ObservedGeneration define the generation of gcu-resource
	ObservedGeneration int `json:"observedGeneration,omitempty"`
	// ResourceStatuses define the status of sub resource
	// +optional
	ResourceStatuses []ResourceStatus `json:"resourceStatuses,omitempty"`
	// Conditions define the condtion of gcu-resource
	Conditions []Condition `json:"conditions,omitempty"`
	// NodesInfo define the cluster nodes information
	NodesInfo map[string]NodeInfo `json:"nodesInfo,omitempty"`
}

// NodeInfo defines the node information
type NodeInfo struct {
	// IsGCUNode define whether the current node is a gcu node
	IsGCUNode bool `json:"isGCUNode,omitempty"`
	// KubeletVersion define current node kubelet version information
	KubeletVersion string `json:"kubeletVersion,omitempty"`
	// ContainerRuntimeVersion define current node container runtime version information
	ContainerRuntimeVersion string `json:"containerRuntimeVersion,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,shortName={"gr", "gcu-res"}
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.readyResource"
// +kubebuilder:printcolumn:name="STATUS",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// GcuResource is the Schema for the gcuresources API
type GcuResource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GcuResourceSpec   `json:"spec,omitempty"`
	Status GcuResourceStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// GcuResourceList contains a list of GcuResource
type GcuResourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GcuResource `json:"items"`
}

func init() {
	SchemeBuilder.Register(&GcuResource{}, &GcuResourceList{})
}
