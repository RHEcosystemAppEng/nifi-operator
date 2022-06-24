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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NifiSpec defines the desired state of Nifi
type NifiSpec struct {
	//+kubebuilder:validation:Minimum=0
	//+kubebuilder:validation:Required
	// Size is the size of the nifi deployment
	Size int32 `json:"size"`

	//+kubebuilder:default:true
	//+kubebuilder:validation:Required
	// UseDefaultCredentials defines if Nifi should be configured with the Single User default Credentials
	UseDefaultCredentials bool `json:"useDefaultCredentials"`
}

// NifiStatus defines the observed state of Nifi
type NifiStatus struct {
	// Nodes are the names of the nifi pods
	Nodes []string `json:"nodes"`

	// Console Route Hostname
	ConsoleRouteHostname string `json:"ConsoleHostname"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Nifi is the Schema for the nifis API
type Nifi struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NifiSpec   `json:"spec,omitempty"`
	Status NifiStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// NifiList contains a list of Nifi
type NifiList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Nifi `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Nifi{}, &NifiList{})
}
