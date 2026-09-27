package tracing

import (
	"os"
	"testing"
)

// lotus v1.37 dropped the Jaeger exporter, so the config fields inherited from
// the Jaeger era have to be translated into the OTLP env vars that
// lib/tracing.SetupOTLPTracing actually reads.
func TestApplyEnvOptsMapsToOTLP(t *testing.T) {
	cases := []struct {
		name         string
		opt          Options
		wantEndpoint string
		wantInsecure string
	}{
		{
			name:         "collector endpoint without scheme implies insecure",
			opt:          Options{CollectorEndpoint: "jaeger.local:4318"},
			wantEndpoint: "jaeger.local:4318",
			wantInsecure: "true",
		},
		{
			name:         "http scheme is stripped and implies insecure",
			opt:          Options{CollectorEndpoint: "http://jaeger.local:4318"},
			wantEndpoint: "jaeger.local:4318",
			wantInsecure: "true",
		},
		{
			name:         "https scheme is stripped and stays secure",
			opt:          Options{CollectorEndpoint: "https://otlp.example.com"},
			wantEndpoint: "otlp.example.com",
			wantInsecure: "",
		},
		{
			name:         "agent endpoint is used as fallback",
			opt:          Options{AgentEndpoint: "agent.local:6831"},
			wantEndpoint: "agent.local:6831",
			wantInsecure: "true",
		},
		{
			name:         "explicit insecure flag",
			opt:          Options{Insecure: true},
			wantEndpoint: "",
			wantInsecure: "true",
		},
		{
			name:         "nothing configured stays untouched",
			opt:          Options{},
			wantEndpoint: "",
			wantInsecure: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			os.Unsetenv(envOTLPExporterEndpoint)
			os.Unsetenv(envOTLPExporterInsecure)
			t.Cleanup(func() {
				os.Unsetenv(envOTLPExporterEndpoint)
				os.Unsetenv(envOTLPExporterInsecure)
			})

			opt := tc.opt
			applyEnvOpts(&opt)

			if got := os.Getenv(envOTLPExporterEndpoint); got != tc.wantEndpoint {
				t.Errorf("endpoint = %q, want %q", got, tc.wantEndpoint)
			}
			if got := os.Getenv(envOTLPExporterInsecure); got != tc.wantInsecure {
				t.Errorf("insecure = %q, want %q", got, tc.wantInsecure)
			}
		})
	}
}

func TestApplyEnvOptsDoesNotClobberExistingEnv(t *testing.T) {
	t.Setenv(envOTLPExporterEndpoint, "preset:4318")

	opt := Options{CollectorEndpoint: "from-config:4318"}
	applyEnvOpts(&opt)

	if got := os.Getenv(envOTLPExporterEndpoint); got != "preset:4318" {
		t.Errorf("endpoint = %q, want the preset value to win", got)
	}
}
