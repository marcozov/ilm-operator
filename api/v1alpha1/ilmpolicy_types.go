/*
Copyright 2021.

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

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ILMPolicySpec defines the desired state of ILMPolicy
type ILMPolicySpec struct {
	// Important: Run "make" to regenerate code after modifying this file
	Body                 string `json:"body,omitempty"`
	ElasticsearchCluster string `json:"elasticsearchCluster,omitempty"`
}

// ILMPolicyStatus defines the observed state of ILMPolicy
type ILMPolicyStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// ILMPolicy is the Schema for the ilmpolicies API
type ILMPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ILMPolicySpec   `json:"spec,omitempty"`
	Status ILMPolicyStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ILMPolicyList contains a list of ILMPolicy
type ILMPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ILMPolicy `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ILMPolicy{}, &ILMPolicyList{})
}
