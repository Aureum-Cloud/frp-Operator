package templates

import (
	"bytes"
	frpv1 "frp-operator/api/v1"
	"frp-operator/internal/constants"
	"html/template"
	"sort"
)

type TemplateData struct {
	ExitServer   *frpv1.ExitServer
	Token        *string
	AdminAPIPort int
	Tunnels      *[]frpv1.Tunnel
}

const CONFIGURATION = `
{{- print -}}
serverAddr = "{{ .ExitServer.Spec.Host }}"
serverPort = {{ .ExitServer.Spec.Port }}

{{- with .Token }}
auth.method = "token"
auth.token = "{{ . }}"
{{- end }}

webServer.addr = "0.0.0.0"
webServer.port = {{ .AdminAPIPort }}

{{- range $tunnel := .Tunnels }}

[[proxies]]
name = "{{ $tunnel.Name }}"
{{- if and $tunnel.Spec.TunnelMapping $tunnel.Spec.TunnelMapping.Protocol }}
type = "{{ $tunnel.Spec.TunnelMapping.Protocol }}"
{{- if and $tunnel.Spec.TunnelMapping.ServiceRef.Name $tunnel.Spec.TunnelMapping.ServiceRef.Namespace }}
localIP = "{{ $tunnel.Spec.TunnelMapping.ServiceRef.Name }}.{{ $tunnel.Spec.TunnelMapping.ServiceRef.Namespace }}.svc"
{{- else }}
localIP = "{{ $tunnel.Spec.TunnelMapping.ServiceRef.Name }}"
{{- end }}
localPort = {{ $tunnel.Spec.TunnelMapping.LocalPort }}
remotePort = {{ $tunnel.Spec.TunnelMapping.RemotePort }}
{{- with $tunnel.Spec.Transport.UseEncryption }}
transport.useEncryption = {{ . }}
{{- end }}
{{- with $tunnel.Spec.Transport.UseCompression }}
transport.useCompression = {{ . }}
{{- end }}
{{- with $tunnel.Spec.Transport.ProxyProtocol }}
transport.proxyProtocolVersion = "{{ . }}"
{{- end }}
{{- with $tunnel.Spec.Transport.BandwidthLimit }}
transport.bandwidthLimit = "{{ . }}"
transport.bandwidthLimitMode = "server"
{{- end }}
{{- end -}}
{{- end -}}
`

func CreateConfiguration(exitServer *frpv1.ExitServer, token string, tunnels []frpv1.Tunnel) (string, error) {
	templateEngine, err := template.New("configuration-template").Parse(CONFIGURATION)
	if err != nil {
		return "", err
	}

	sort.Slice(tunnels[:], func(i, j int) bool {
		return tunnels[i].Name < tunnels[j].Name
	})

	var buffer bytes.Buffer
	err = templateEngine.Execute(&buffer, TemplateData{
		ExitServer:   exitServer,
		Token:        &token,
		AdminAPIPort: constants.AdminAPIPort,
		Tunnels:      &tunnels,
	})
	if err != nil {
		return "", err
	}

	return buffer.String(), nil
}
