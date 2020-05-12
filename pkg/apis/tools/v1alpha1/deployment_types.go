// Copyright 2019 The Kubernetes Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type GenericContainerSpec struct {
	Enabled *bool    `json:"enabled,omitempty"`
	Name    *string  `json:"name,omitempty"`
	Image   *string  `json:"image,omitempty"`
	Command []string `json:"command,omitempty"`
	Args    []string `json:"args,omitempty"`
}

var (
	ContainerEnvVarKeyWATCHNAMESPACE = "WATCH_NAMESPACE"
	ContainerEnvVarKeyPODNAME        = "POD_NAME"
	ContainerEnvVarKeyOPERATORNAME   = "OPERATOR_NAME"

	DefaultPodServiceAccountName = "ham-deploy"
)

var (
	DefaultDeployableEnablement       = true
	DefaultDeployableContainerName    = "deployable"
	DefaultDeployableContainerImage   = "quay.io/hybridappio/ham-deployable-operator"
	DefaultDeployableContainerCommand = []string{"ham-deployable-operator"}
)

type DeployableOperatorSpec struct {
	GenericContainerSpec `json:",inline"`
}

var (
	DefaultAssmeblerEnablement       = true
	DefaultAssemblerContainerName    = "assembler"
	DefaultAssemblerContainerImage   = "quay.io/hybridappio/ham-application-assembler"
	DefaultAssemblerContainerCommand = []string{"ham-application-assembler"}
)

type ApplicationAssemblerSpec struct {
	GenericContainerSpec `json:",inline"`
}

var (
	DefaultDiscovererEnablement       = true
	DefaultDiscovererContainerName    = "discoverer"
	DefaultDiscovererContainerImage   = "quay.io/hybridappio/ham-resource-discoverer"
	DefaultDiscovererContainerCommand = []string{"ham-resource-discoverer"}

	DefaultPodVolumeNameHubConnection = "hub-connection-config"
)

var (
	ContainerEnvVarKeyCLUSTERNAME      = "CLUSTERNAME"
	ContainerEnvVarKeyCLUSTERNAMESPACE = "CLUSTERNAMESPACE"
	ContainerEnvVarKeyHUBKUBECONFIG    = "HUBCLUSTERCONFIGFILE"
)

type HubConnectionConfig struct {
	KubeConfig *string                     `json:"kubeconfig,omitempty"`
	MountPath  string                      `json:"mountpath"`
	SecretRef  corev1.LocalObjectReference `json:"secretRef"`
}

type ResourceDiscovererSpec struct {
	GenericContainerSpec `json:",inline"`
	ClusterName          string               `json:"clustername"`
	ClusterNamespace     string               `json:"clusternamespace"`
	HubConnectionConfig  *HubConnectionConfig `json:"hubconfig,omitempty"`
}

type CoreSpec struct {
	DeployableOperatorSpec *DeployableOperatorSpec `json:"deployable,omitempty"`
}

type ToolsSpec struct {
	ApplicationAssemblerSpec *ApplicationAssemblerSpec `json:"assembler,omitempty"`
	ResourceDiscovererSpec   *ResourceDiscovererSpec   `json:"discoverer,omitempty"`
}

// DeploymentSpec defines the desired state of Deployment
type DeploymentSpec struct {
	CoreSpec  *CoreSpec  `json:"core,omitempty"`
	ToolsSpec *ToolsSpec `json:"tools,omitempty"`
}

// DeploymentStatus defines the observed state of Deployment
type DeploymentStatus struct {
	corev1.PodStatus `json:",inline"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Deployment is the Schema for the deployments API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=deployments,scope=Namespaced
type Deployment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DeploymentSpec   `json:"spec,omitempty"`
	Status DeploymentStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DeploymentList contains a list of Deployment
type DeploymentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Deployment `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Deployment{}, &DeploymentList{})
}
