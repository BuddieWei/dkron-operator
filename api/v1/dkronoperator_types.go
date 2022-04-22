/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DkronOperatorSpec defines the desired state of DkronOperator
type DkronOperatorSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// DkronConfig is the configuration for the dkron instances
	DkronConfig *DkronConfig `json:"dkronConfig,omitempty"`

	// Replicas is the number of replicas of the dkron instances.
	// Default is 3
	// +kubebuilder:validation:Minimum=1
	// +required
	Replicas int32 `json:"replicas"`

	// Namespace is the namespace which will be used to create the dkron statefulset
	// +optional
	Namespace string `json:"namespace"`

	// Registry is the image repository for dkron instances
	// +optional
	Registry string `json:"registry,omitempty"`

	// Image is the image for dkron instances
	// +kubebuilder:default=dkron/dkron
	// +optional
	ImageName string `json:"imageName,omitempty"`

	// ImageTag is the image tag for dkron instances, can be empty
	// +kubebuilder:default=v3.1.11
	// +optional
	ImageTag string `json:"imageTag,omitempty"`

	// ImagePullPolicy is the image pull policy for dkron instances
	// +kubebuilder:default=Always
	// +optional
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`

	// ImagePullSecrets is the image pull secrets for dkron instances
	// +optional
	ImagePullSecrets []string `json:"imagePullSecrets,omitempty"`

	// Resources is the resource requirements for dkron instances
	// +optional
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`

	// InitialDelaySeconds is the initial delay seconds for dkron instances
	// +kubebuilder:default=30
	// +optional
	LivenessInitialDelaySeconds int32 `json:"livenessInitialDelaySeconds,omitempty"`

	// ReadinessInitialDelaySeconds is the initial delay seconds for dkron instances
	// +kubebuilder:default=30
	// +optional
	ReadinessInitialDelaySeconds int32 `json:"readinessInitialDelaySeconds,omitempty"`

	// Affinity is the affinity for dkron instances
	// +optional
	Affinity *corev1.Affinity `json:"affinity,omitempty"`

	// Tolerations is the tolerations for dkron instances
	// +optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// NodeSelector is the node selector for dkron instances
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// ServiceAccountName is the service account name for dkron instances
	// If not specified, the service account "controller-manager" will be used
	// +kubebuilder:default=controller-manager
	// +required
	ServiceAccountName string `json:"serviceAccountName,omitempty"`

	// HostAliases is the hostAliases for dkron instances
	// HostAliases is an optional list of hosts and IPs that will be injected into the pod's hosts
	// file if specified. This is only valid for non-hostNetwork pods.
	// +optional
	HostAliases []corev1.HostAlias `json:"hostAliases,omitempty"`

	// Annotations is the annotations for dkron instances
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`

	// StorageClass is the storage class for dkron instances
	// +optional
	StorageClass string `json:"storageClass,omitempty"`

	// StorageSize is the storage size for dkron instances
	// +optional
	StorageSize string `json:"storageSize,omitempty"`

	// EnableIngress is the flag to enable ingress
	// +kubebuilder:default=false
	// +optional
	EnableIngress bool `json:"enableIngress,omitempty"`

	// BaseDomain is the base domain for dkron instances
	// If EnableIngress is true, BaseDomain will be needed
	// +optional
	BaseDomain string `json:"baseDomain,omitempty"`

	// IngressName is the name of the ingress
	// If EnableIngress is true, IngressName will be needed
	// ingress host: {IngressName}.{BaseDomain}
	// +kubebuilder:default=dkron-server
	// +optional
	IngressName string `json:"ingressName,omitempty"`
}

// DkronOperatorStatus defines the observed state of DkronOperator
type DkronOperatorStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// doc: https://dkron.io/usage/clustering/

	ServiceUri string `json:"serviceUri,omitempty"`
	IngressUri string `json:"ingressUri,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// DkronOperator is the Schema for the dkronoperators API
type DkronOperator struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DkronOperatorSpec   `json:"spec,omitempty"`
	Status DkronOperatorStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// DkronOperatorList contains a list of DkronOperator
type DkronOperatorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DkronOperator `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DkronOperator{}, &DkronOperatorList{})
}

type DkronConfig struct {

	// EnableDocker is the enable prometheus metrics
	// +kubebuilder:default=true
	EnablePrometheus bool `json:"enablePrometheus"`

	// BootstrapExpect is the bootstrap expect for dkron instances
	// +kubebuilder:default=3
	BootstrapExpect int `json:"bootstrapExpect"`

	// DisableUsageStats is the disable usage stats for dkron instances
	// +kubebuilder:default=false
	DisableUsageStats bool `json:"disableUsageStats"`

	// DataDir is the data dir for dkron instances
	// +kubebuilder:default=/root/dkron.data
	DataDir string `json:"dataDir"`
}
