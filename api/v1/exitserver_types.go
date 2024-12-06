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

// ExitServerSpec defines the desired state of ExitServer
type ExitServerSpec struct {
	Host           string         `json:"host"`
	Port           int            `json:"port"`
	Authentication Authentication `json:"authentication"`
}

type Authentication struct {
	Token *Token `json:"token"`
}

type Token struct {
	Secret SecretKeyRef `json:"secretKeyRef"`
}

type SecretKeyRef struct {
	Name string `json:"name"`
	Key  string `json:"key"`
}

// ExitServerStatus defines the observed state of ExitServer
type ExitServerStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// ExitServer is the Schema for the exitservers API
type ExitServer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ExitServerSpec   `json:"spec,omitempty"`
	Status ExitServerStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ExitServerList contains a list of ExitServer
type ExitServerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ExitServer `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ExitServer{}, &ExitServerList{})
}
