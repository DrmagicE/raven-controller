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

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// EndpointSpec defines the desired state of Endpoint
type EndpointSpec struct {
	// NodePool is the name of the node pool that the endpoint belongs to.
	NodePool string `json:"nodePool"`
	// NodeName is the name of the node that the endpoint belongs to.
	NodeName      string            `json:"nodeName"`
	PrivateIP     string            `json:"privateIP"`
	PublicIP      string            `json:"publicIP"`
	NATEnabled    bool              `json:"natEnabled,omitempty"`
	Backend       string            `json:"backend"`
	BackendConfig map[string]string `json:"backendConfig,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:printcolumn:name="NodeName",type=string,JSONPath=`.spec.nodeName`
//+kubebuilder:printcolumn:name="PrivateIP",type=string,JSONPath=`.spec.privateIP`
//+kubebuilder:printcolumn:name="PublicIP",type=string,JSONPath=`.spec.publicIP`

// Endpoint is the Schema for the endpoints API
type Endpoint struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec EndpointSpec `json:"spec,omitempty"`
}

//+kubebuilder:object:root=true

// EndpointList contains a list of Endpoint
type EndpointList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Endpoint `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Endpoint{}, &EndpointList{})
}
