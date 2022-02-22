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

// GatewaySpec defines the desired state of Gateway
type GatewaySpec struct {
	// NodePool is the name of the nodepool which the Gateway belongs to.
	NodePool string `json:"nodePool"`
	// Topologies represents which topologies the Gateway is participated in.
	Topologies []Topology `json:"topologies"`
}

// Topology represents the VPN topology.
// Raven will only establish VPN between the gateways in the same topology.
type Topology struct {
	// Name is the name of the topology.
	Name string `json:"name"`
}

// GatewayStatus defines the observed state of Gateway
type GatewayStatus struct {
	// Subnets contains all the subnets in the gateway.
	Subnets []string `json:"subnets,omitempty"`
	// ActiveEndpoint is the reference of the active endpoint.
	ActiveEndpoint *EndpointSpec `json:"activeEndpoint,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Cluster
//+kubebuilder:printcolumn:name="ActiveEndpoint",type=string,JSONPath=`.status.activeEndpoint.nodeName`

// Gateway is the Schema for the gateways API
type Gateway struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GatewaySpec   `json:"spec,omitempty"`
	Status GatewayStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// GatewayList contains a list of Gateway
type GatewayList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Gateway `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Gateway{}, &GatewayList{})
}
