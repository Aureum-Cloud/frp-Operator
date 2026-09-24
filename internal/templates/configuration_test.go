package templates

import (
	"strings"
	"testing"

	frpv1 "frp-operator/api/v1"

	"github.com/pelletier/go-toml/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func ptr[T any](value T) *T {
	return &value
}

func exitServer() *frpv1.ExitServer {
	return &frpv1.ExitServer{
		Spec: frpv1.ExitServerSpec{
			Host: "frps.example.com",
			Port: 7000,
		},
	}
}

func TestCreateConfigurationRendersTCPProxy(t *testing.T) {
	tunnels := []frpv1.Tunnel{{
		ObjectMeta: metav1.ObjectMeta{Name: "tcp-tunnel"},
		Spec: frpv1.TunnelSpec{
			ExitServer: "guest-frps",
			TCP: &frpv1.TCP{
				ServiceRef: frpv1.ServiceRef{Name: "my-svc", Namespace: ptr("my-ns")},
				LocalPort:  1234,
				RemotePort: 1234,
			},
			Transport: &frpv1.Transport{UseEncryption: true},
		},
	}}

	configuration, err := CreateConfiguration(exitServer(), "token", tunnels)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := strings.Join([]string{
		`[[proxies]]`,
		`name = "tcp-tunnel"`,
		`type = "tcp"`,
		`localIP = "my-svc.my-ns.svc"`,
		`localPort = 1234`,
		`remotePort = 1234`,
		`transport.useEncryption = true`,
	}, "\n")

	if !strings.Contains(configuration, expected) {
		t.Fatalf("expected configuration to contain:\n%s\ngot:\n%s", expected, configuration)
	}
}

func TestCreateConfigurationRendersHTTPProxyWithPlugin(t *testing.T) {
	tunnels := []frpv1.Tunnel{{
		ObjectMeta: metav1.ObjectMeta{Name: "guest-k8s-traefik"},
		Spec: frpv1.TunnelSpec{
			ExitServer: "guest-frps",
			HTTP: &frpv1.HTTP{
				CustomDomains: []string{"*.example.com"},
				Plugin: &frpv1.Plugin{
					Type:       "http2http",
					ServiceRef: frpv1.ServiceRef{Name: "traefik", Namespace: ptr("traefik")},
					LocalPort:  80,
				},
			},
		},
	}}

	configuration, err := CreateConfiguration(exitServer(), "token", tunnels)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := strings.Join([]string{
		`[[proxies]]`,
		`name = "guest-k8s-traefik"`,
		`type = "http"`,
		`customDomains = ["*.example.com"]`,
		``,
		`[proxies.plugin]`,
		`type = "http2http"`,
		`localAddr = "traefik.traefik.svc:80"`,
	}, "\n")

	if !strings.Contains(configuration, expected) {
		t.Fatalf("expected configuration to contain:\n%s\ngot:\n%s", expected, configuration)
	}
}

func TestCreateConfigurationRendersHTTPProxyWithServiceRef(t *testing.T) {
	tunnels := []frpv1.Tunnel{{
		ObjectMeta: metav1.ObjectMeta{Name: "http-tunnel"},
		Spec: frpv1.TunnelSpec{
			ExitServer: "guest-frps",
			HTTP: &frpv1.HTTP{
				CustomDomains: []string{"app.example.com", "www.example.com"},
				ServiceRef:    &frpv1.ServiceRef{Name: "web"},
				LocalPort:     ptr(8080),
			},
			Transport: &frpv1.Transport{UseCompression: true},
		},
	}}

	configuration, err := CreateConfiguration(exitServer(), "token", tunnels)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := strings.Join([]string{
		`[[proxies]]`,
		`name = "http-tunnel"`,
		`type = "http"`,
		`customDomains = ["app.example.com", "www.example.com"]`,
		`localIP = "web"`,
		`localPort = 8080`,
		`transport.useCompression = true`,
	}, "\n")

	if !strings.Contains(configuration, expected) {
		t.Fatalf("expected configuration to contain:\n%s\ngot:\n%s", expected, configuration)
	}
}

func TestCreateConfigurationEscapesTokenAndHost(t *testing.T) {
	server := exitServer()
	server.Spec.Host = `frps."evil".org`

	configuration, err := CreateConfiguration(server, `to"ken`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(configuration, `serverAddr = "frps.\"evil\".org"`) {
		t.Fatalf("expected escaped host, got:\n%s", configuration)
	}
	if !strings.Contains(configuration, `auth.token = "to\"ken"`) {
		t.Fatalf("expected escaped token, got:\n%s", configuration)
	}
}

func TestCreateConfigurationRendersHTTPRoutingOptions(t *testing.T) {
	tunnels := []frpv1.Tunnel{{
		ObjectMeta: metav1.ObjectMeta{Name: "http-routing"},
		Spec: frpv1.TunnelSpec{
			ExitServer: "guest-frps",
			HTTP: &frpv1.HTTP{
				Subdomain:         ptr("portal"),
				Locations:         []string{"/", "/api"},
				HTTPUser:          ptr("alice"),
				HTTPPassword:      ptr(`p"ass`),
				HostHeaderRewrite: ptr("backend.example.com"),
				RouteByHTTPUser:   ptr("alice"),
				RequestHeaders: map[string]string{
					"x-from-where": "frp",
					"x-trace":      `a"b`,
				},
				ResponseHeaders: map[string]string{"x-proxy": "frp"},
				ServiceRef:      &frpv1.ServiceRef{Name: "web", Namespace: ptr("apps")},
				LocalPort:       ptr(8080),
			},
		},
	}}

	configuration, err := CreateConfiguration(exitServer(), "token", tunnels)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedLines := []string{
		`subdomain = "portal"`,
		`locations = ["/", "/api"]`,
		`httpUser = "alice"`,
		`httpPassword = "p\"ass"`,
		`hostHeaderRewrite = "backend.example.com"`,
		`routeByHTTPUser = "alice"`,
		`requestHeaders.set."x-from-where" = "frp"`,
		`requestHeaders.set."x-trace" = "a\"b"`,
		`responseHeaders.set."x-proxy" = "frp"`,
		`localIP = "web.apps.svc"`,
		`localPort = 8080`,
	}
	for _, expected := range expectedLines {
		if !strings.Contains(configuration, expected) {
			t.Errorf("expected configuration to contain %q, got:\n%s", expected, configuration)
		}
	}
	if strings.Contains(configuration, "customDomains =") {
		t.Errorf("did not expect customDomains for a subdomain-only proxy, got:\n%s", configuration)
	}
}

func TestCreateConfigurationRendersPluginOptions(t *testing.T) {
	tunnels := []frpv1.Tunnel{{
		ObjectMeta: metav1.ObjectMeta{Name: "plugin"},
		Spec: frpv1.TunnelSpec{
			ExitServer: "guest-frps",
			HTTP: &frpv1.HTTP{
				CustomDomains: []string{"plugin.example.com"},
				Plugin: &frpv1.Plugin{
					Type:              "http2https",
					ServiceRef:        frpv1.ServiceRef{Name: "secure-web"},
					LocalPort:         8443,
					HostHeaderRewrite: ptr("origin.example.com"),
					RequestHeaders:    map[string]string{"x-forwarded-by": "frp"},
				},
			},
		},
	}}

	configuration, err := CreateConfiguration(exitServer(), "token", tunnels)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := strings.Join([]string{
		`[proxies.plugin]`,
		`type = "http2https"`,
		`localAddr = "secure-web:8443"`,
		`hostHeaderRewrite = "origin.example.com"`,
		`requestHeaders.set."x-forwarded-by" = "frp"`,
	}, "\n")
	if !strings.Contains(configuration, expected) {
		t.Fatalf("expected configuration to contain:\n%s\ngot:\n%s", expected, configuration)
	}
	if strings.Contains(configuration, "localIP =") || strings.Contains(configuration, "localPort =") {
		t.Fatalf("plugin proxy must not render plain backend fields, got:\n%s", configuration)
	}
}

func TestCreateConfigurationRendersAllTransportOptions(t *testing.T) {
	tunnels := []frpv1.Tunnel{{
		ObjectMeta: metav1.ObjectMeta{Name: "transport"},
		Spec: frpv1.TunnelSpec{
			ExitServer: "guest-frps",
			TCP: &frpv1.TCP{
				ServiceRef: frpv1.ServiceRef{Name: "service"},
				LocalPort:  8080,
				RemotePort: 18080,
			},
			Transport: &frpv1.Transport{
				UseEncryption:      true,
				UseCompression:     true,
				ProxyProtocol:      ptr("v2"),
				BandwidthLimit:     ptr("10MB"),
				BandwidthLimitMode: ptr("client"),
			},
		},
	}}

	configuration, err := CreateConfiguration(exitServer(), "token", tunnels)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, expected := range []string{
		`transport.useEncryption = true`,
		`transport.useCompression = true`,
		`transport.proxyProtocolVersion = "v2"`,
		`transport.bandwidthLimit = "10MB"`,
		`transport.bandwidthLimitMode = "client"`,
	} {
		if !strings.Contains(configuration, expected) {
			t.Errorf("expected configuration to contain %q, got:\n%s", expected, configuration)
		}
	}
}

func TestCreateConfigurationSortsProxiesByName(t *testing.T) {
	newTunnel := func(name string) frpv1.Tunnel {
		return frpv1.Tunnel{
			ObjectMeta: metav1.ObjectMeta{Name: name},
			Spec: frpv1.TunnelSpec{
				ExitServer: "guest-frps",
				TCP:        &frpv1.TCP{ServiceRef: frpv1.ServiceRef{Name: "service"}, LocalPort: 80, RemotePort: 80},
			},
		}
	}

	configuration, err := CreateConfiguration(exitServer(), "token", []frpv1.Tunnel{newTunnel("z-last"), newTunnel("a-first")})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	first := strings.Index(configuration, `name = "a-first"`)
	last := strings.Index(configuration, `name = "z-last"`)
	if first < 0 || last < 0 || first >= last {
		t.Fatalf("expected proxies sorted by name, got:\n%s", configuration)
	}
}

func TestCreateConfigurationRendersV071HTTPConfigurationExactly(t *testing.T) {
	tunnels := []frpv1.Tunnel{{
		ObjectMeta: metav1.ObjectMeta{Name: "kitchen-sink"},
		Spec: frpv1.TunnelSpec{
			ExitServer: "guest-frps",
			HTTP: &frpv1.HTTP{
				CustomDomains:     []string{"app.example.com"},
				Subdomain:         ptr("app"),
				Locations:         []string{"/", "/api"},
				HTTPUser:          ptr("alice"),
				HTTPPassword:      ptr("secret"),
				HostHeaderRewrite: ptr("backend.example.com"),
				RouteByHTTPUser:   ptr("alice"),
				RequestHeaders:    map[string]string{"z-last": "1", "a-first": "2"},
				ResponseHeaders:   map[string]string{"foo": "bar"},
				ServiceRef:        &frpv1.ServiceRef{Name: "web", Namespace: ptr("apps")},
				LocalPort:         ptr(8080),
			},
			Transport: &frpv1.Transport{
				UseEncryption:      true,
				UseCompression:     true,
				ProxyProtocol:      ptr("v2"),
				BandwidthLimit:     ptr("10MB"),
				BandwidthLimitMode: ptr("client"),
			},
		},
	}}

	configuration, err := CreateConfiguration(exitServer(), "token", tunnels)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := strings.Join([]string{
		`serverAddr = "frps.example.com"`,
		`serverPort = 7000`,
		`auth.method = "token"`,
		`auth.token = "token"`,
		``,
		`webServer.addr = "0.0.0.0"`,
		`webServer.port = 7400`,
		``,
		`[[proxies]]`,
		`name = "kitchen-sink"`,
		`type = "http"`,
		`customDomains = ["app.example.com"]`,
		`subdomain = "app"`,
		`locations = ["/", "/api"]`,
		`httpUser = "alice"`,
		`httpPassword = "secret"`,
		`hostHeaderRewrite = "backend.example.com"`,
		`routeByHTTPUser = "alice"`,
		`requestHeaders.set."a-first" = "2"`,
		`requestHeaders.set."z-last" = "1"`,
		`responseHeaders.set."foo" = "bar"`,
		`localIP = "web.apps.svc"`,
		`localPort = 8080`,
		`transport.useEncryption = true`,
		`transport.useCompression = true`,
		`transport.proxyProtocolVersion = "v2"`,
		`transport.bandwidthLimit = "10MB"`,
		`transport.bandwidthLimitMode = "client"`,
	}, "\n")
	if configuration != expected {
		t.Fatalf("unexpected v0.71 configuration (-want +got):\nwant:\n%s\n\ngot:\n%s", expected, configuration)
	}
}

func TestCreateConfigurationRendersGlobalConfigurationWithoutProxies(t *testing.T) {
	configuration, err := CreateConfiguration(exitServer(), "token", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := strings.Join([]string{
		`serverAddr = "frps.example.com"`,
		`serverPort = 7000`,
		`auth.method = "token"`,
		`auth.token = "token"`,
		``,
		`webServer.addr = "0.0.0.0"`,
		`webServer.port = 7400`,
	}, "\n")
	if configuration != expected {
		t.Fatalf("unexpected global configuration:\nwant:\n%s\n\ngot:\n%s", expected, configuration)
	}
}

func TestCreateConfigurationOmitsUnsetOptionalFields(t *testing.T) {
	tunnels := []frpv1.Tunnel{{
		ObjectMeta: metav1.ObjectMeta{Name: "plain"},
		Spec: frpv1.TunnelSpec{
			ExitServer: "guest-frps",
			TCP:        &frpv1.TCP{ServiceRef: frpv1.ServiceRef{Name: "svc"}, LocalPort: 80, RemotePort: 8080},
			Transport:  &frpv1.Transport{UseEncryption: false, UseCompression: false},
		},
	}}
	configuration, err := CreateConfiguration(exitServer(), "token", tunnels)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, unexpected := range []string{
		"transport.useEncryption",
		"transport.useCompression",
		"transport.proxyProtocolVersion",
		"transport.bandwidthLimit",
		"transport.bandwidthLimitMode",
		"[proxies.plugin]",
	} {
		if strings.Contains(configuration, unexpected) {
			t.Errorf("did not expect %q in configuration:\n%s", unexpected, configuration)
		}
	}
}

func TestCreateConfigurationDefaultsBandwidthLimitModeToClient(t *testing.T) {
	tunnels := []frpv1.Tunnel{{
		ObjectMeta: metav1.ObjectMeta{Name: "limited"},
		Spec: frpv1.TunnelSpec{
			ExitServer: "guest-frps",
			TCP:        &frpv1.TCP{ServiceRef: frpv1.ServiceRef{Name: "svc"}, LocalPort: 80, RemotePort: 8080},
			Transport:  &frpv1.Transport{BandwidthLimit: ptr("2MB")},
		},
	}}

	configuration, err := CreateConfiguration(exitServer(), "token", tunnels)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "transport.bandwidthLimit = \"2MB\"\ntransport.bandwidthLimitMode = \"client\""
	if !strings.Contains(configuration, expected) {
		t.Fatalf("expected default client-side bandwidth limit:\n%s", configuration)
	}
}

func TestCreateConfigurationEscapesV071HTTPValues(t *testing.T) {
	tunnels := []frpv1.Tunnel{{
		ObjectMeta: metav1.ObjectMeta{Name: `proxy"name`},
		Spec: frpv1.TunnelSpec{
			ExitServer: "guest-frps",
			HTTP: &frpv1.HTTP{
				CustomDomains:  []string{`app."example.com`, `path\example.com`},
				Locations:      []string{`/say/"hello"`, `/back\slash`},
				RequestHeaders: map[string]string{`x."quoted`: `value"quoted`},
				ServiceRef:     &frpv1.ServiceRef{Name: "web"},
				LocalPort:      ptr(8080),
			},
		},
	}}

	configuration, err := CreateConfiguration(exitServer(), "token", tunnels)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, expected := range []string{
		`name = "proxy\"name"`,
		`customDomains = ["app.\"example.com", "path\\example.com"]`,
		`locations = ["/say/\"hello\"", "/back\\slash"]`,
		`requestHeaders.set."x.\"quoted" = "value\"quoted"`,
	} {
		if !strings.Contains(configuration, expected) {
			t.Errorf("expected escaped value %q in configuration:\n%s", expected, configuration)
		}
	}
}

func TestCreateConfigurationProducesValidTOMLWithControlCharacters(t *testing.T) {
	tunnels := []frpv1.Tunnel{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "plugin\ttunnel"},
			Spec: frpv1.TunnelSpec{
				ExitServer: "guest-frps",
				HTTP: &frpv1.HTTP{
					CustomDomains: []string{"app.example.com"},
					Plugin: &frpv1.Plugin{
						Type:       "http2http",
						ServiceRef: frpv1.ServiceRef{Name: "web"},
						LocalPort:  8080,
					},
				},
				Annotations: map[string]string{"owner\x01": "platform\nteam"},
				Metadatas:   map[string]string{"environment": "prod\x7f"},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "tcp-tunnel"},
			Spec: frpv1.TunnelSpec{
				ExitServer: "guest-frps",
				TCP: &frpv1.TCP{
					ServiceRef: frpv1.ServiceRef{Name: "api"},
					LocalPort:  8080,
					RemotePort: 18080,
				},
			},
		},
	}

	configuration, err := CreateConfiguration(exitServer(), "token\x02", tunnels)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed map[string]any
	if err := toml.Unmarshal([]byte(configuration), &parsed); err != nil {
		t.Fatalf("configuration is not valid TOML: %v\n%s", err, configuration)
	}

	proxies, ok := parsed["proxies"].([]any)
	if !ok || len(proxies) != 2 {
		t.Fatalf("expected two parsed proxies, got %#v", parsed["proxies"])
	}
	first, ok := proxies[0].(map[string]any)
	if !ok {
		t.Fatalf("expected first proxy to be a table, got %#v", proxies[0])
	}
	if first["name"] != "plugin\ttunnel" {
		t.Fatalf("expected control characters to round-trip, got %#v", first["name"])
	}
	if _, ok := first["plugin"].(map[string]any); !ok {
		t.Fatalf("expected plugin table on first proxy, got %#v", first["plugin"])
	}
	if _, ok := first["annotations"].(map[string]any); !ok {
		t.Fatalf("expected annotations table on first proxy, got %#v", first["annotations"])
	}
	second, ok := proxies[1].(map[string]any)
	if !ok || second["name"] != "tcp-tunnel" || second["remotePort"] != int64(18080) {
		t.Fatalf("expected second proxy fields to remain top-level, got %#v", proxies[1])
	}
}

// proxySection returns the rendered `[[proxies]]` block of the first proxy.
func proxySection(t *testing.T, configuration string) string {
	t.Helper()

	index := strings.Index(configuration, "[[proxies]]")
	if index < 0 {
		t.Fatalf("expected a proxy section in configuration:\n%s", configuration)
	}
	return configuration[index:]
}

func TestCreateConfigurationRendersUDPProxy(t *testing.T) {
	tunnels := []frpv1.Tunnel{{
		ObjectMeta: metav1.ObjectMeta{Name: "udp-tunnel"},
		Spec: frpv1.TunnelSpec{
			ExitServer: "guest-frps",
			UDP: &frpv1.UDP{
				ServiceRef: frpv1.ServiceRef{Name: "dns", Namespace: ptr("infra")},
				LocalPort:  53,
				RemotePort: 5353,
			},
		},
	}}

	configuration, err := CreateConfiguration(exitServer(), "token", tunnels)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := strings.Join([]string{
		`[[proxies]]`,
		`name = "udp-tunnel"`,
		`type = "udp"`,
		`localIP = "dns.infra.svc"`,
		`localPort = 53`,
		`remotePort = 5353`,
	}, "\n")
	if proxySection(t, configuration) != expected {
		t.Fatalf("unexpected udp proxy:\nwant:\n%s\n\ngot:\n%s", expected, proxySection(t, configuration))
	}
}

func TestCreateConfigurationRendersHTTPSProxy(t *testing.T) {
	tunnels := []frpv1.Tunnel{{
		ObjectMeta: metav1.ObjectMeta{Name: "https-tunnel"},
		Spec: frpv1.TunnelSpec{
			ExitServer: "guest-frps",
			HTTPS: &frpv1.HTTPS{
				CustomDomains: []string{"secure.example.com"},
				ServiceRef:    &frpv1.ServiceRef{Name: "web", Namespace: ptr("apps")},
				LocalPort:     ptr(8443),
			},
		},
	}}

	configuration, err := CreateConfiguration(exitServer(), "token", tunnels)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := strings.Join([]string{
		`[[proxies]]`,
		`name = "https-tunnel"`,
		`type = "https"`,
		`customDomains = ["secure.example.com"]`,
		`localIP = "web.apps.svc"`,
		`localPort = 8443`,
	}, "\n")
	if proxySection(t, configuration) != expected {
		t.Fatalf("unexpected https proxy:\nwant:\n%s\n\ngot:\n%s", expected, proxySection(t, configuration))
	}
}

func TestCreateConfigurationRendersHTTPSProxyWithHTTPS2HTTPPlugin(t *testing.T) {
	tunnels := []frpv1.Tunnel{{
		ObjectMeta: metav1.ObjectMeta{Name: "https-plugin"},
		Spec: frpv1.TunnelSpec{
			ExitServer: "guest-frps",
			HTTPS: &frpv1.HTTPS{
				Subdomain: ptr("secure"),
				Plugin: &frpv1.Plugin{
					Type:              "https2http",
					ServiceRef:        frpv1.ServiceRef{Name: "web"},
					LocalPort:         80,
					HostHeaderRewrite: ptr("backend.example.com"),
					CrtPath:           ptr("/certs/tls.crt"),
					KeyPath:           ptr("/certs/tls.key"),
					RequestHeaders:    map[string]string{"x-from-where": "frp"},
				},
			},
		},
	}}

	configuration, err := CreateConfiguration(exitServer(), "token", tunnels)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := strings.Join([]string{
		`[[proxies]]`,
		`name = "https-plugin"`,
		`type = "https"`,
		`subdomain = "secure"`,
		``,
		`[proxies.plugin]`,
		`type = "https2http"`,
		`localAddr = "web:80"`,
		`hostHeaderRewrite = "backend.example.com"`,
		`crtPath = "/certs/tls.crt"`,
		`keyPath = "/certs/tls.key"`,
		`requestHeaders.set."x-from-where" = "frp"`,
	}, "\n")
	if proxySection(t, configuration) != expected {
		t.Fatalf("unexpected https2http proxy:\nwant:\n%s\n\ngot:\n%s", expected, proxySection(t, configuration))
	}
}

func TestCreateConfigurationRendersTCPMuxProxy(t *testing.T) {
	tunnels := []frpv1.Tunnel{{
		ObjectMeta: metav1.ObjectMeta{Name: "mux-tunnel"},
		Spec: frpv1.TunnelSpec{
			ExitServer: "guest-frps",
			TCPMux: &frpv1.TCPMux{
				Multiplexer:     "httpconnect",
				CustomDomains:   []string{"mux.example.com"},
				HTTPUser:        ptr("alice"),
				HTTPPassword:    ptr("secret"),
				RouteByHTTPUser: ptr("alice"),
				ServiceRef:      frpv1.ServiceRef{Name: "backend"},
				LocalPort:       8080,
			},
		},
	}}

	configuration, err := CreateConfiguration(exitServer(), "token", tunnels)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := strings.Join([]string{
		`[[proxies]]`,
		`name = "mux-tunnel"`,
		`type = "tcpmux"`,
		`multiplexer = "httpconnect"`,
		`customDomains = ["mux.example.com"]`,
		`httpUser = "alice"`,
		`httpPassword = "secret"`,
		`routeByHTTPUser = "alice"`,
		`localIP = "backend"`,
		`localPort = 8080`,
	}, "\n")
	if proxySection(t, configuration) != expected {
		t.Fatalf("unexpected tcpmux proxy:\nwant:\n%s\n\ngot:\n%s", expected, proxySection(t, configuration))
	}
}

func TestCreateConfigurationRendersSTCPProxy(t *testing.T) {
	tunnels := []frpv1.Tunnel{{
		ObjectMeta: metav1.ObjectMeta{Name: "stcp-tunnel"},
		Spec: frpv1.TunnelSpec{
			ExitServer: "guest-frps",
			STCP: &frpv1.SecretProxy{
				SecretKey:  "abcdefg",
				AllowUsers: []string{"user1", "user2"},
				ServiceRef: frpv1.ServiceRef{Name: "ssh", Namespace: ptr("infra")},
				LocalPort:  22,
			},
		},
	}}

	configuration, err := CreateConfiguration(exitServer(), "token", tunnels)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := strings.Join([]string{
		`[[proxies]]`,
		`name = "stcp-tunnel"`,
		`type = "stcp"`,
		`secretKey = "abcdefg"`,
		`allowUsers = ["user1", "user2"]`,
		`localIP = "ssh.infra.svc"`,
		`localPort = 22`,
	}, "\n")
	if proxySection(t, configuration) != expected {
		t.Fatalf("unexpected stcp proxy:\nwant:\n%s\n\ngot:\n%s", expected, proxySection(t, configuration))
	}
}

func TestCreateConfigurationRendersSUDPProxy(t *testing.T) {
	tunnels := []frpv1.Tunnel{{
		ObjectMeta: metav1.ObjectMeta{Name: "sudp-tunnel"},
		Spec: frpv1.TunnelSpec{
			ExitServer: "guest-frps",
			SUDP: &frpv1.SecretProxy{
				SecretKey:  "abcdefg",
				ServiceRef: frpv1.ServiceRef{Name: "dns"},
				LocalPort:  53,
			},
		},
	}}

	configuration, err := CreateConfiguration(exitServer(), "token", tunnels)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := strings.Join([]string{
		`[[proxies]]`,
		`name = "sudp-tunnel"`,
		`type = "sudp"`,
		`secretKey = "abcdefg"`,
		`localIP = "dns"`,
		`localPort = 53`,
	}, "\n")
	if proxySection(t, configuration) != expected {
		t.Fatalf("unexpected sudp proxy:\nwant:\n%s\n\ngot:\n%s", expected, proxySection(t, configuration))
	}
}

func TestCreateConfigurationRendersXTCPProxyWithNatTraversal(t *testing.T) {
	tunnels := []frpv1.Tunnel{{
		ObjectMeta: metav1.ObjectMeta{Name: "xtcp-tunnel"},
		Spec: frpv1.TunnelSpec{
			ExitServer: "guest-frps",
			XTCP: &frpv1.XTCP{
				SecretProxy: frpv1.SecretProxy{
					SecretKey:  "abcdefg",
					AllowUsers: []string{"*"},
					ServiceRef: frpv1.ServiceRef{Name: "ssh"},
					LocalPort:  22,
				},
				NatTraversal: &frpv1.NatTraversal{DisableAssistedAddrs: true},
			},
		},
	}}

	configuration, err := CreateConfiguration(exitServer(), "token", tunnels)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := strings.Join([]string{
		`[[proxies]]`,
		`name = "xtcp-tunnel"`,
		`type = "xtcp"`,
		`secretKey = "abcdefg"`,
		`allowUsers = ["*"]`,
		`localIP = "ssh"`,
		`localPort = 22`,
		``,
		`[proxies.natTraversal]`,
		`disableAssistedAddrs = true`,
	}, "\n")
	if proxySection(t, configuration) != expected {
		t.Fatalf("unexpected xtcp proxy:\nwant:\n%s\n\ngot:\n%s", expected, proxySection(t, configuration))
	}
}

func TestCreateConfigurationRendersCommonProxyOptions(t *testing.T) {
	tunnels := []frpv1.Tunnel{{
		ObjectMeta: metav1.ObjectMeta{Name: "common"},
		Spec: frpv1.TunnelSpec{
			ExitServer:  "guest-frps",
			Enabled:     ptr(false),
			Metadatas:   map[string]string{"z-last": "1", "a-first": "2"},
			Annotations: map[string]string{"k8s.io/owner": "team"},
			LoadBalancer: &frpv1.LoadBalancer{
				Group:    "web",
				GroupKey: ptr("123"),
			},
			HealthCheck: &frpv1.HealthCheck{
				Type:            "http",
				Path:            ptr("/status"),
				TimeoutSeconds:  ptr(3),
				MaxFailed:       ptr(3),
				IntervalSeconds: ptr(10),
				HTTPHeaders:     []frpv1.HTTPHeader{{Name: "x-from-where", Value: "frp"}},
			},
			TCP: &frpv1.TCP{
				ServiceRef: frpv1.ServiceRef{Name: "web"},
				LocalPort:  8080,
				RemotePort: 18080,
			},
		},
	}}

	configuration, err := CreateConfiguration(exitServer(), "token", tunnels)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := strings.Join([]string{
		`[[proxies]]`,
		`name = "common"`,
		`type = "tcp"`,
		`localIP = "web"`,
		`localPort = 8080`,
		`remotePort = 18080`,
		`enabled = false`,
		`loadBalancer.group = "web"`,
		`loadBalancer.groupKey = "123"`,
		`healthCheck.type = "http"`,
		`healthCheck.timeoutSeconds = 3`,
		`healthCheck.maxFailed = 3`,
		`healthCheck.intervalSeconds = 10`,
		`healthCheck.path = "/status"`,
		`healthCheck.httpHeaders = [{ name = "x-from-where", value = "frp" }]`,
		`metadatas."a-first" = "2"`,
		`metadatas."z-last" = "1"`,
		``,
		`[proxies.annotations]`,
		`"k8s.io/owner" = "team"`,
	}, "\n")
	if proxySection(t, configuration) != expected {
		t.Fatalf("unexpected common options:\nwant:\n%s\n\ngot:\n%s", expected, proxySection(t, configuration))
	}
}

func TestCreateConfigurationKeepsCommonFieldsOutOfPluginTable(t *testing.T) {
	tunnels := []frpv1.Tunnel{{
		ObjectMeta: metav1.ObjectMeta{Name: "scope"},
		Spec: frpv1.TunnelSpec{
			ExitServer:  "guest-frps",
			Metadatas:   map[string]string{"env": "prod"},
			HealthCheck: &frpv1.HealthCheck{Type: "tcp"},
			HTTP: &frpv1.HTTP{
				CustomDomains: []string{"app.example.com"},
				Plugin: &frpv1.Plugin{
					Type:       "http2http",
					ServiceRef: frpv1.ServiceRef{Name: "web"},
					LocalPort:  80,
				},
			},
			Transport: &frpv1.Transport{UseEncryption: true},
		},
	}}

	configuration, err := CreateConfiguration(exitServer(), "token", tunnels)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	pluginTable := strings.Index(configuration, "[proxies.plugin]")
	if pluginTable < 0 {
		t.Fatalf("expected a plugin table, got:\n%s", configuration)
	}
	for _, commonField := range []string{
		"transport.useEncryption",
		"healthCheck.type",
		`metadatas."env"`,
	} {
		index := strings.Index(configuration, commonField)
		if index < 0 {
			t.Errorf("expected %q in configuration:\n%s", commonField, configuration)
			continue
		}
		if index > pluginTable {
			t.Errorf("%q must be rendered before [proxies.plugin] to stay in the proxy table:\n%s", commonField, configuration)
		}
	}
}

func TestCreateConfigurationRendersPluginVariants(t *testing.T) {
	testCases := []struct {
		name     string
		plugin   frpv1.Plugin
		expected []string
		omitted  []string
	}{
		{
			name: "http_proxy",
			plugin: frpv1.Plugin{
				Type:         "http_proxy",
				HTTPUser:     ptr("abc"),
				HTTPPassword: ptr("abc"),
			},
			expected: []string{`type = "http_proxy"`, `httpUser = "abc"`, `httpPassword = "abc"`},
			omitted:  []string{"localAddr"},
		},
		{
			name: "socks5",
			plugin: frpv1.Plugin{
				Type:     "socks5",
				Username: ptr("abc"),
				Password: ptr("abc"),
			},
			expected: []string{`type = "socks5"`, `username = "abc"`, `password = "abc"`},
			omitted:  []string{"localAddr"},
		},
		{
			name: "static_file",
			plugin: frpv1.Plugin{
				Type:         "static_file",
				LocalPath:    ptr("/var/www/blog"),
				StripPrefix:  ptr("static"),
				HTTPUser:     ptr("abc"),
				HTTPPassword: ptr("abc"),
			},
			expected: []string{`type = "static_file"`, `localPath = "/var/www/blog"`, `stripPrefix = "static"`},
			omitted:  []string{"localAddr"},
		},
		{
			name: "unix_domain_socket",
			plugin: frpv1.Plugin{
				Type:     "unix_domain_socket",
				UnixPath: ptr("/var/run/docker.sock"),
			},
			expected: []string{`type = "unix_domain_socket"`, `unixPath = "/var/run/docker.sock"`},
			omitted:  []string{"localAddr"},
		},
		{
			name: "tls2raw",
			plugin: frpv1.Plugin{
				Type:       "tls2raw",
				ServiceRef: frpv1.ServiceRef{Name: "web"},
				LocalPort:  80,
				CrtPath:    ptr("/certs/tls.crt"),
				KeyPath:    ptr("/certs/tls.key"),
			},
			expected: []string{`type = "tls2raw"`, `localAddr = "web:80"`, `crtPath = "/certs/tls.crt"`, `keyPath = "/certs/tls.key"`},
		},
		{
			name:     "virtual_net",
			plugin:   frpv1.Plugin{Type: "virtual_net"},
			expected: []string{`type = "virtual_net"`},
			omitted:  []string{"localAddr"},
		},
		{
			name: "https2https",
			plugin: frpv1.Plugin{
				Type:        "https2https",
				ServiceRef:  frpv1.ServiceRef{Name: "web"},
				LocalPort:   443,
				CrtPath:     ptr("/certs/tls.crt"),
				KeyPath:     ptr("/certs/tls.key"),
				EnableHTTP2: ptr(true),
			},
			expected: []string{`type = "https2https"`, `localAddr = "web:443"`, `enableHTTP2 = true`},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			plugin := testCase.plugin
			tunnels := []frpv1.Tunnel{{
				ObjectMeta: metav1.ObjectMeta{Name: "plugin"},
				Spec: frpv1.TunnelSpec{
					ExitServer: "guest-frps",
					TCP: &frpv1.TCP{
						ServiceRef: frpv1.ServiceRef{Name: "unused"},
						LocalPort:  1,
						RemotePort: 6000,
						Plugin:     &plugin,
					},
				},
			}}

			configuration, err := CreateConfiguration(exitServer(), "token", tunnels)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			for _, expected := range testCase.expected {
				if !strings.Contains(configuration, expected) {
					t.Errorf("expected %q in configuration:\n%s", expected, configuration)
				}
			}
			for _, omitted := range testCase.omitted {
				if strings.Contains(configuration, omitted) {
					t.Errorf("did not expect %q in configuration:\n%s", omitted, configuration)
				}
			}
			if strings.Contains(configuration, "localIP =") {
				t.Errorf("plugin backed proxy must not render localIP:\n%s", configuration)
			}
		})
	}
}
