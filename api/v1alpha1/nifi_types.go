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

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NifiConsoleSpec contains the Nifi Console configuration
type NifiConsoleSpec struct {
	//+kubebuilder:validation:Required
	Expose bool `json:"expose"`

	//+kubebuilder:validation:Required
	//+kubebuilder:validation:Pattern=^http|HTTP|https|HTTPS$
	//+kubebuilder:default:http
	Protocol string `json:"protocol"`

	//+kubebuilder:validation:Optional
	RouteHostname string `json:"routeHostname"`
}

// NifiSpec defines the desired state of Nifi
type NifiSpec struct {
	//+kubebuilder:validation:Minimum=0
	//+kubebuilder:validation:Required
	//+kubebuilder:default:0
	// Size is the size of the nifi deployment
	Size int32 `json:"size"`

	//+kubebuilder:validation:Optional
	// Image the container image for the Nifi deployment
	Image string `json:"image"`

	//+kubebuilder:default:true
	//+kubebuilder:validation:Required
	// UseDefaultCredentials defines if Nifi should be configured with the Single User default Credentials
	UseDefaultCredentials bool `json:"useDefaultCredentials"`

	//+kubebuilder:validation:Required
	Console NifiConsoleSpec `json:"console"`

	//+kubebuilder:validation:Optional
	Resources *corev1.ResourceRequirements `json:"resources"`
}

// NifiStatus defines the observed state of Nifi
type NifiStatus struct {
	// Nodes are the names of the nifi pods
	Nodes []string `json:"nodes"`

	// UI Route reference
	UIRoute string `json:"uiRoute"`
}

// Nifi is the Schema for the nifis API
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type Nifi struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NifiSpec   `json:"spec,omitempty"`
	Status NifiStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// NifiList contains a list of Nifi
type NifiList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Nifi `json:"items"`
}

// NifiRegistrySpec defines the desired state of Nifi Registry
type NifiRegistrySpec struct {
	//+kubebuilder:validation:Optional
	// Image the container image for the Nifi deployment
	Image string `json:"image"`

	//+kubebuilder:validation:Required
	Expose bool `json:"expose"`
}

// NifiStatus defines the observed state of Nifi
type NifiRegistryStatus struct {
	// UI Route reference
	UIRoute string `json:"uiRoute"`
}

// NifiRegistry is the Schema for the nifi registries API
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type NifiRegistry struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NifiRegistrySpec   `json:"spec,omitempty"`
	Status NifiRegistryStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// NifiList contains a list of Nifi
type NifiRegistryList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NifiRegistry `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Nifi{}, &NifiList{})
	SchemeBuilder.Register(&NifiRegistry{}, &NifiRegistryList{})
}
