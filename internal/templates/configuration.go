package templates

import (
	"bytes"
	"fmt"
	frpv1 "frp-operator/api/v1"
	"frp-operator/internal/constants"
	"sort"
	"strconv"
	"strings"
	"text/template"
)

type TemplateData struct {
	ExitServer   *frpv1.ExitServer
	Token        *string
	AdminAPIPort int
	Tunnels      *[]frpv1.Tunnel
}

const CONFIGURATION = `
{{- print -}}
serverAddr = {{ quote .ExitServer.Spec.Host }}
serverPort = {{ .ExitServer.Spec.Port }}

{{- with .Token }}
auth.method = "token"
auth.token = {{ quote . }}
{{- end }}

webServer.addr = "0.0.0.0"
webServer.port = {{ .AdminAPIPort }}

{{- range $tunnel := .Tunnels }}

[[proxies]]
name = {{ quote $tunnel.Name }}
{{- if $tunnel.Spec.TCP }}
type = "tcp"
{{- if not $tunnel.Spec.TCP.Plugin }}
localIP = {{ serviceAddress $tunnel.Spec.TCP.ServiceRef }}
localPort = {{ $tunnel.Spec.TCP.LocalPort }}
{{- end }}
remotePort = {{ $tunnel.Spec.TCP.RemotePort }}
{{- else if $tunnel.Spec.UDP }}
type = "udp"
localIP = {{ serviceAddress $tunnel.Spec.UDP.ServiceRef }}
localPort = {{ $tunnel.Spec.UDP.LocalPort }}
remotePort = {{ $tunnel.Spec.UDP.RemotePort }}
{{- else if $tunnel.Spec.HTTP }}
type = "http"
{{- with $tunnel.Spec.HTTP.CustomDomains }}
customDomains = {{ stringArray $tunnel.Spec.HTTP.CustomDomains }}
{{- end }}
{{- with $tunnel.Spec.HTTP.Subdomain }}
subdomain = {{ quote . }}
{{- end }}
{{- with $tunnel.Spec.HTTP.Locations }}
locations = {{ stringArray . }}
{{- end }}
{{- with $tunnel.Spec.HTTP.HTTPUser }}
httpUser = {{ quote . }}
{{- end }}
{{- with $tunnel.Spec.HTTP.HTTPPassword }}
httpPassword = {{ quote . }}
{{- end }}
{{- with $tunnel.Spec.HTTP.HostHeaderRewrite }}
hostHeaderRewrite = {{ quote . }}
{{- end }}
{{- with $tunnel.Spec.HTTP.RouteByHTTPUser }}
routeByHTTPUser = {{ quote . }}
{{- end }}
{{- range $key, $value := $tunnel.Spec.HTTP.RequestHeaders }}
requestHeaders.set.{{ tomlKey $key }} = {{ quote $value }}
{{- end }}
{{- range $key, $value := $tunnel.Spec.HTTP.ResponseHeaders }}
responseHeaders.set.{{ tomlKey $key }} = {{ quote $value }}
{{- end }}
{{- if not $tunnel.Spec.HTTP.Plugin }}
localIP = {{ serviceAddress $tunnel.Spec.HTTP.ServiceRef }}
localPort = {{ $tunnel.Spec.HTTP.LocalPort }}
{{- end }}
{{- else if $tunnel.Spec.HTTPS }}
type = "https"
{{- with $tunnel.Spec.HTTPS.CustomDomains }}
customDomains = {{ stringArray . }}
{{- end }}
{{- with $tunnel.Spec.HTTPS.Subdomain }}
subdomain = {{ quote . }}
{{- end }}
{{- if not $tunnel.Spec.HTTPS.Plugin }}
localIP = {{ serviceAddress $tunnel.Spec.HTTPS.ServiceRef }}
localPort = {{ $tunnel.Spec.HTTPS.LocalPort }}
{{- end }}
{{- else if $tunnel.Spec.TCPMux }}
type = "tcpmux"
multiplexer = {{ quote $tunnel.Spec.TCPMux.Multiplexer }}
{{- with $tunnel.Spec.TCPMux.CustomDomains }}
customDomains = {{ stringArray . }}
{{- end }}
{{- with $tunnel.Spec.TCPMux.Subdomain }}
subdomain = {{ quote . }}
{{- end }}
{{- with $tunnel.Spec.TCPMux.HTTPUser }}
httpUser = {{ quote . }}
{{- end }}
{{- with $tunnel.Spec.TCPMux.HTTPPassword }}
httpPassword = {{ quote . }}
{{- end }}
{{- with $tunnel.Spec.TCPMux.RouteByHTTPUser }}
routeByHTTPUser = {{ quote . }}
{{- end }}
localIP = {{ serviceAddress $tunnel.Spec.TCPMux.ServiceRef }}
localPort = {{ $tunnel.Spec.TCPMux.LocalPort }}
{{- else if $tunnel.Spec.STCP }}
type = "stcp"
secretKey = {{ quote $tunnel.Spec.STCP.SecretKey }}
{{- with $tunnel.Spec.STCP.AllowUsers }}
allowUsers = {{ stringArray . }}
{{- end }}
{{- if not $tunnel.Spec.STCP.Plugin }}
localIP = {{ serviceAddress $tunnel.Spec.STCP.ServiceRef }}
localPort = {{ $tunnel.Spec.STCP.LocalPort }}
{{- end }}
{{- else if $tunnel.Spec.SUDP }}
type = "sudp"
secretKey = {{ quote $tunnel.Spec.SUDP.SecretKey }}
{{- with $tunnel.Spec.SUDP.AllowUsers }}
allowUsers = {{ stringArray . }}
{{- end }}
{{- if not $tunnel.Spec.SUDP.Plugin }}
localIP = {{ serviceAddress $tunnel.Spec.SUDP.ServiceRef }}
localPort = {{ $tunnel.Spec.SUDP.LocalPort }}
{{- end }}
{{- else if $tunnel.Spec.XTCP }}
type = "xtcp"
secretKey = {{ quote $tunnel.Spec.XTCP.SecretKey }}
{{- with $tunnel.Spec.XTCP.AllowUsers }}
allowUsers = {{ stringArray . }}
{{- end }}
{{- if not $tunnel.Spec.XTCP.Plugin }}
localIP = {{ serviceAddress $tunnel.Spec.XTCP.ServiceRef }}
localPort = {{ $tunnel.Spec.XTCP.LocalPort }}
{{- end }}
{{- end }}
{{- if isDisabled $tunnel.Spec.Enabled }}
enabled = false
{{- end }}
{{- with $tunnel.Spec.Transport }}
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
{{- with $tunnel.Spec.Transport.BandwidthLimitMode }}
transport.bandwidthLimitMode = "{{ . }}"
{{- else }}
transport.bandwidthLimitMode = "client"
{{- end }}
{{- end }}
{{- end -}}
{{- with $tunnel.Spec.LoadBalancer }}
loadBalancer.group = {{ quote .Group }}
{{- with .GroupKey }}
loadBalancer.groupKey = {{ quote . }}
{{- end }}
{{- end }}
{{- with $tunnel.Spec.HealthCheck }}
healthCheck.type = {{ quote .Type }}
{{- with .TimeoutSeconds }}
healthCheck.timeoutSeconds = {{ . }}
{{- end }}
{{- with .MaxFailed }}
healthCheck.maxFailed = {{ . }}
{{- end }}
{{- with .IntervalSeconds }}
healthCheck.intervalSeconds = {{ . }}
{{- end }}
{{- with .Path }}
healthCheck.path = {{ quote . }}
{{- end }}
{{- with .HTTPHeaders }}
healthCheck.httpHeaders = {{ headerArray . }}
{{- end }}
{{- end }}
{{- range $key, $value := $tunnel.Spec.Metadatas }}
metadatas.{{ tomlKey $key }} = {{ quote $value }}
{{- end }}
{{- with pluginOf $tunnel }}

[proxies.plugin]
type = {{ quote .Type }}
{{- if needsLocalAddr .Type }}
localAddr = {{ localAddress .ServiceRef .LocalPort }}
{{- end }}
{{- with .HostHeaderRewrite }}
hostHeaderRewrite = {{ quote . }}
{{- end }}
{{- with .HTTPUser }}
httpUser = {{ quote . }}
{{- end }}
{{- with .HTTPPassword }}
httpPassword = {{ quote . }}
{{- end }}
{{- with .Username }}
username = {{ quote . }}
{{- end }}
{{- with .Password }}
password = {{ quote . }}
{{- end }}
{{- with .LocalPath }}
localPath = {{ quote . }}
{{- end }}
{{- with .StripPrefix }}
stripPrefix = {{ quote . }}
{{- end }}
{{- with .UnixPath }}
unixPath = {{ quote . }}
{{- end }}
{{- with .CrtPath }}
crtPath = {{ quote . }}
{{- end }}
{{- with .KeyPath }}
keyPath = {{ quote . }}
{{- end }}
{{- with .EnableHTTP2 }}
enableHTTP2 = {{ . }}
{{- end }}
{{- range $key, $value := .RequestHeaders }}
requestHeaders.set.{{ tomlKey $key }} = {{ quote $value }}
{{- end }}
{{- end }}
{{- with $tunnel.Spec.XTCP }}
{{- with .NatTraversal }}

[proxies.natTraversal]
disableAssistedAddrs = {{ .DisableAssistedAddrs }}
{{- end }}
{{- end }}
{{- with $tunnel.Spec.Annotations }}

[proxies.annotations]
{{- range $key, $value := . }}
{{ tomlKey $key }} = {{ quote $value }}
{{- end }}
{{- end }}
{{- end -}}
`

func CreateConfiguration(exitServer *frpv1.ExitServer, token string, tunnels []frpv1.Tunnel) (string, error) {
	templateEngine, err := template.New("configuration-template").Funcs(template.FuncMap{
		"localAddress":   localAddress,
		"quote":          tomlString,
		"serviceAddress": serviceAddress,
		"stringArray":    stringArray,
		"tomlKey":        tomlString,
		"headerArray":    headerArray,
		"isDisabled":     isDisabled,
		"needsLocalAddr": needsLocalAddr,
		"pluginOf":       pluginOf,
	}).Parse(CONFIGURATION)
	if err != nil {
		return "", err
	}

	sorted := make([]frpv1.Tunnel, len(tunnels))
	copy(sorted, tunnels)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Name < sorted[j].Name
	})

	var buffer bytes.Buffer
	err = templateEngine.Execute(&buffer, TemplateData{
		ExitServer:   exitServer,
		Token:        &token,
		AdminAPIPort: constants.AdminAPIPort,
		Tunnels:      &sorted,
	})
	if err != nil {
		return "", err
	}

	return buffer.String(), nil
}

// tomlString encodes a value as a TOML basic string. strconv.Quote is not used
// directly because it emits Go-only escapes such as \xNN for control characters,
// which the TOML specification does not accept.
func tomlString(value string) string {
	var builder strings.Builder
	builder.WriteByte('"')
	for _, r := range value {
		switch r {
		case '"':
			builder.WriteString(`\"`)
		case '\\':
			builder.WriteString(`\\`)
		case '\b':
			builder.WriteString(`\b`)
		case '\f':
			builder.WriteString(`\f`)
		case '\n':
			builder.WriteString(`\n`)
		case '\r':
			builder.WriteString(`\r`)
		case '\t':
			builder.WriteString(`\t`)
		default:
			// TOML requires control characters other than the ones above to be
			// written using the \uXXXX or \UXXXXXXXX escape forms.
			if r < 0x20 || r == 0x7f {
				builder.WriteString(fmt.Sprintf(`\u%04X`, r))
			} else {
				builder.WriteRune(r)
			}
		}
	}
	builder.WriteByte('"')
	return builder.String()
}

func serviceAddress(serviceRef frpv1.ServiceRef) string {
	return tomlString(rawServiceAddress(serviceRef))
}

func rawServiceAddress(serviceRef frpv1.ServiceRef) string {
	address := serviceRef.Name
	if serviceRef.Namespace != nil {
		address += "." + *serviceRef.Namespace + ".svc"
	}

	return address
}

func localAddress(serviceRef frpv1.ServiceRef, port int) string {
	return tomlString(rawServiceAddress(serviceRef) + ":" + strconv.Itoa(port))
}

func stringArray(values []string) string {
	quoted := make([]string, len(values))
	for index, value := range values {
		quoted[index] = tomlString(value)
	}
	return "[" + strings.Join(quoted, ", ") + "]"
}

func headerArray(headers []frpv1.HTTPHeader) string {
	entries := make([]string, len(headers))
	for index, header := range headers {
		entries[index] = "{ name = " + tomlString(header.Name) + ", value = " + tomlString(header.Value) + " }"
	}
	return "[" + strings.Join(entries, ", ") + "]"
}

func isDisabled(enabled *bool) bool {
	return enabled != nil && !*enabled
}

// needsLocalAddr reports whether the frp client plugin forwards to a local address.
func needsLocalAddr(pluginType string) bool {
	switch pluginType {
	case "http2http", "http2https", "https2http", "https2https", "tls2raw":
		return true
	default:
		return false
	}
}

func pluginOf(tunnel frpv1.Tunnel) *frpv1.Plugin {
	switch {
	case tunnel.Spec.TCP != nil:
		return tunnel.Spec.TCP.Plugin
	case tunnel.Spec.HTTP != nil:
		return tunnel.Spec.HTTP.Plugin
	case tunnel.Spec.HTTPS != nil:
		return tunnel.Spec.HTTPS.Plugin
	case tunnel.Spec.STCP != nil:
		return tunnel.Spec.STCP.Plugin
	case tunnel.Spec.SUDP != nil:
		return tunnel.Spec.SUDP.Plugin
	case tunnel.Spec.XTCP != nil:
		return tunnel.Spec.XTCP.Plugin
	default:
		return nil
	}
}
