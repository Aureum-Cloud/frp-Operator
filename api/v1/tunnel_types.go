/*
Copyright 2026 Aureum Cloud, N-Bit, Niek Berenschot.

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
// +kubebuilder:validation:XValidation:rule="[has(self.tcp), has(self.udp), has(self.http), has(self.https), has(self.tcpmux), has(self.stcp), has(self.xtcp), has(self.sudp)].filter(x, x).size() == 1",message="exactly one proxy type must be set"
type TunnelSpec struct {
	ExitServer string `json:"exitServer"`

	// +optional
	Enabled *bool `json:"enabled,omitempty"`

	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`

	// +optional
	Metadatas map[string]string `json:"metadatas,omitempty"`

	// +optional
	LoadBalancer *LoadBalancer `json:"loadBalancer,omitempty"`

	// +optional
	HealthCheck *HealthCheck `json:"healthCheck,omitempty"`

	// +optional
	TCP *TCP `json:"tcp,omitempty"`

	// +optional
	HTTP *HTTP `json:"http,omitempty"`

	// +optional
	UDP *UDP `json:"udp,omitempty"`

	// +optional
	HTTPS *HTTPS `json:"https,omitempty"`

	// +optional
	TCPMux *TCPMux `json:"tcpmux,omitempty"`

	// +optional
	STCP *SecretProxy `json:"stcp,omitempty"`

	// +optional
	XTCP *XTCP `json:"xtcp,omitempty"`

	// +optional
	SUDP *SecretProxy `json:"sudp,omitempty"`

	// +optional
	Transport *Transport `json:"transport,omitempty"`
}

// +kubebuilder:validation:XValidation:rule="has(self.plugin) != (has(self.serviceRef) && self.serviceRef.name != ” && has(self.localPort) && self.localPort > 0)",message="either plugin or both serviceRef and localPort must be set"
type TCP struct {
	// +optional
	ServiceRef ServiceRef `json:"serviceRef,omitempty"`
	// +optional
	LocalPort  int `json:"localPort,omitempty"`
	RemotePort int `json:"remotePort"`

	// +optional
	Plugin *Plugin `json:"plugin,omitempty"`
}

type UDP struct {
	ServiceRef ServiceRef `json:"serviceRef"`
	LocalPort  int        `json:"localPort"`
	RemotePort int        `json:"remotePort"`
}

// HTTP exposes the tunnel as a vhost based HTTP proxy on the exit server.
// +kubebuilder:validation:XValidation:rule="has(self.plugin) != (has(self.serviceRef) && self.serviceRef.name != ” && has(self.localPort) && self.localPort > 0)",message="either plugin or both serviceRef and localPort must be set"
// +kubebuilder:validation:XValidation:rule="size(self.customDomains) > 0 || has(self.subdomain)",message="at least one custom domain or a subdomain must be set"
type HTTP struct {
	// +optional
	CustomDomains []string `json:"customDomains,omitempty"`

	// +optional
	Subdomain *string `json:"subdomain,omitempty"`

	// +optional
	Locations []string `json:"locations,omitempty"`

	// +optional
	HTTPUser *string `json:"httpUser,omitempty"`

	// +optional
	HTTPPassword *string `json:"httpPassword,omitempty"`

	// +optional
	HostHeaderRewrite *string `json:"hostHeaderRewrite,omitempty"`

	// +optional
	RequestHeaders map[string]string `json:"requestHeaders,omitempty"`

	// +optional
	ResponseHeaders map[string]string `json:"responseHeaders,omitempty"`

	// +optional
	RouteByHTTPUser *string `json:"routeByHTTPUser,omitempty"`

	// +optional
	ServiceRef *ServiceRef `json:"serviceRef,omitempty"`

	// +optional
	LocalPort *int `json:"localPort,omitempty"`

	// +optional
	Plugin *Plugin `json:"plugin,omitempty"`
}

// Plugin configures an frpc client plugin instead of a plain local address.
// +kubebuilder:validation:XValidation:rule="!(self.type in ['http2http', 'http2https', 'https2http', 'https2https', 'tls2raw']) || (has(self.serviceRef) && self.serviceRef.name != ” && has(self.localPort) && self.localPort > 0)",message="this plugin type requires both serviceRef and localPort"
type Plugin struct {
	// +kubebuilder:validation:Enum=http2http;http2https;http_proxy;https2http;https2https;socks5;static_file;unix_domain_socket;tls2raw;virtual_net
	Type string `json:"type"`

	// +optional
	ServiceRef ServiceRef `json:"serviceRef,omitempty"`

	// +optional
	LocalPort int `json:"localPort,omitempty"`

	// +optional
	HostHeaderRewrite *string `json:"hostHeaderRewrite,omitempty"`

	// +optional
	RequestHeaders map[string]string `json:"requestHeaders,omitempty"`

	// +optional
	HTTPUser *string `json:"httpUser,omitempty"`

	// +optional
	HTTPPassword *string `json:"httpPassword,omitempty"`

	// +optional
	Username *string `json:"username,omitempty"`

	// +optional
	Password *string `json:"password,omitempty"`

	// +optional
	LocalPath *string `json:"localPath,omitempty"`

	// +optional
	StripPrefix *string `json:"stripPrefix,omitempty"`

	// +optional
	UnixPath *string `json:"unixPath,omitempty"`

	// +optional
	CrtPath *string `json:"crtPath,omitempty"`

	// +optional
	KeyPath *string `json:"keyPath,omitempty"`

	// +optional
	EnableHTTP2 *bool `json:"enableHTTP2,omitempty"`
}

// +kubebuilder:validation:XValidation:rule="has(self.plugin) != (has(self.serviceRef) && self.serviceRef.name != ” && has(self.localPort) && self.localPort > 0)",message="either plugin or both serviceRef and localPort must be set"
// +kubebuilder:validation:XValidation:rule="size(self.customDomains) > 0 || has(self.subdomain)",message="at least one custom domain or a subdomain must be set"
type HTTPS struct {
	CustomDomains []string    `json:"customDomains,omitempty"`
	Subdomain     *string     `json:"subdomain,omitempty"`
	ServiceRef    *ServiceRef `json:"serviceRef,omitempty"`
	LocalPort     *int        `json:"localPort,omitempty"`
	Plugin        *Plugin     `json:"plugin,omitempty"`
}

type TCPMux struct {
	CustomDomains   []string `json:"customDomains,omitempty"`
	Subdomain       *string  `json:"subdomain,omitempty"`
	HTTPUser        *string  `json:"httpUser,omitempty"`
	HTTPPassword    *string  `json:"httpPassword,omitempty"`
	RouteByHTTPUser *string  `json:"routeByHTTPUser,omitempty"`
	// +kubebuilder:validation:Enum=httpconnect
	Multiplexer string     `json:"multiplexer"`
	ServiceRef  ServiceRef `json:"serviceRef"`
	LocalPort   int        `json:"localPort"`
}

// +kubebuilder:validation:XValidation:rule="has(self.plugin) != (has(self.serviceRef) && self.serviceRef.name != ” && has(self.localPort) && self.localPort > 0)",message="either plugin or both serviceRef and localPort must be set"
type SecretProxy struct {
	SecretKey  string   `json:"secretKey"`
	AllowUsers []string `json:"allowUsers,omitempty"`
	// +optional
	ServiceRef ServiceRef `json:"serviceRef,omitempty"`
	// +optional
	LocalPort int     `json:"localPort,omitempty"`
	Plugin    *Plugin `json:"plugin,omitempty"`
}

type XTCP struct {
	SecretProxy  `json:",inline"`
	NatTraversal *NatTraversal `json:"natTraversal,omitempty"`
}

type NatTraversal struct {
	// +optional
	DisableAssistedAddrs bool `json:"disableAssistedAddrs,omitempty"`
}

type LoadBalancer struct {
	Group string `json:"group"`
	// +optional
	GroupKey *string `json:"groupKey,omitempty"`
}

type HTTPHeader struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type HealthCheck struct {
	// +kubebuilder:validation:Enum=tcp;http
	Type string `json:"type"`
	// +optional
	TimeoutSeconds *int `json:"timeoutSeconds,omitempty"`
	// +optional
	MaxFailed *int `json:"maxFailed,omitempty"`
	// +optional
	IntervalSeconds *int `json:"intervalSeconds,omitempty"`
	// +optional
	Path *string `json:"path,omitempty"`
	// +optional
	HTTPHeaders []HTTPHeader `json:"httpHeaders,omitempty"`
}

type ServiceRef struct {
	Name      string  `json:"name"`
	Namespace *string `json:"namespace,omitempty"`
}

type Transport struct {
	// +optional
	UseEncryption bool `json:"useEncryption,omitempty"`

	// +optional
	UseCompression bool `json:"useCompression,omitempty"`

	// +kubebuilder:validation:Enum=v1;v2
	// +optional
	ProxyProtocol *string `json:"proxyProtocol,omitempty"`

	// +kubebuilder:validation:Pattern=^\d+(\.\d+)?(KB|MB)$
	// +optional
	BandwidthLimit *string `json:"bandwidthLimit,omitempty"`

	// +kubebuilder:validation:Enum=client;server
	// +optional
	BandwidthLimitMode *string `json:"bandwidthLimitMode,omitempty"`
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
