/*
Copyright 2024 Aureum Cloud, N-Bit, Niek Berenschot.

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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TunnelSpec defines the desired state of Tunnel
type TunnelSpec struct {
	ExitServer string     `json:"exitServer"`
	TCP        *TCP       `json:"tcp"`
	Transport  *Transport `json:"transport"`
}

type TCP struct {
	ServiceRef ServiceRef `json:"serviceRef"`
	LocalPort  int        `json:"localPort"`
	RemotePort int        `json:"remotePort"`
}

type ServiceRef struct {
	Name      string  `json:"name"`
	Namespace *string `json:"namespace,omitempty"`
}

type Transport struct {
	UseEncryption  bool `json:"useEncryption"`
	UseCompression bool `json:"useCompression"`

	// +kubebuilder:validation:Enum=v1;v2
	// +optional
	ProxyProtocol *string `json:"proxyProtocol"`

	// +kubebuilder:validation:Pattern=^\d+(KB|MB)$
	// +optional
	BandwidthLimit *string `json:"bandwidthLimit"`
}

// TunnelStatus defines the observed state of Tunnel
type TunnelStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Tunnel is the Schema for the tunnels API
type Tunnel struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TunnelSpec   `json:"spec,omitempty"`
	Status TunnelStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// TunnelList contains a list of Tunnel
type TunnelList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Tunnel `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Tunnel{}, &TunnelList{})
}
