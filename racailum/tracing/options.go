// GENERATED, DO NOT EDIT
package tracing

import (
	"os"
	"strings"
)

const defaultName = "londobell"

// lotus v1.37 removed the Jaeger exporter from lib/tracing; only OTLP remains
// (lotus/lib/tracing/setup.go: SetupOTLPTracing reads these two variables).
const (
	envOTLPExporterEndpoint = "LOTUS_OTEL_EXPORTER_ENDPOINT"
	envOTLPExporterInsecure = "LOTUS_OTEL_EXPORTER_INSECURE"
)

// The *Endpoint / *Host / *Port / Jaeger* config fields below are kept as-is so
// that existing config files keep parsing. They are translated to the OTLP
// environment variables that lotus now reads.
func applyEnvOpts(opt *Options) {
	// CollectorEndpoint is the Jaeger collector, AgentEndpoint the Jaeger agent;
	// both map onto the single OTLP endpoint knob.
	endpoint := opt.CollectorEndpoint
	if endpoint == "" {
		endpoint = opt.AgentEndpoint
	}

	if endpoint != "" {
		insecure := true
		if strings.HasPrefix(endpoint, "https://") {
			// otlptracehttp.WithEndpoint wants host:port, without a scheme.
			endpoint = strings.TrimPrefix(endpoint, "https://")
			insecure = false
		}
		endpoint = strings.TrimPrefix(endpoint, "http://")

		setEnvIfUnset(envOTLPExporterEndpoint, endpoint)
		if insecure {
			setEnvIfUnset(envOTLPExporterInsecure, "true")
		}
	}

	if opt.Insecure {
		setEnvIfUnset(envOTLPExporterInsecure, "true")
	}
}

func setEnvIfUnset(key, value string) {
	if os.Getenv(key) == "" {
		os.Setenv(key, value)
	}
}

type Options struct {
	Enable  bool
	Name    string
	Sampler *float64

	// Insecure forces plaintext OTLP export. It is implied when the endpoint
	// carries no https:// scheme.
	Insecure bool

	CollectorEndpoint string
	AgentEndpoint     string
	AgentHost         string
	AgentPort         string
	JaegerUser        string
	JaegerCred        string
}

func DefaultOptions() Options {
	return Options{
		Enable: true,
		Name:   defaultName,
	}
}
