// GENERATED, DO NOT EDIT
package tracing

import (
	"os"
)

const defaultName = "londobell"

const (
	envCollectorEndpoint = "LOTUS_JAEGER_COLLECTOR_ENDPOINT"
	envAgentEndpoint     = "LOTUS_JAEGER_AGENT_ENDPOINT"
	envAgentHost         = "LOTUS_JAEGER_AGENT_HOST"
	envAgentPort         = "LOTUS_JAEGER_AGENT_PORT"
	envJaegerUser        = "LOTUS_JAEGER_USERNAME"
	envJaegerCred        = "LOTUS_JAEGER_PASSWORD"
)

func applyEnvOpts(opt *Options) {
	if opt.CollectorEndpoint != "" {
		if set := os.Getenv(envCollectorEndpoint); set == "" {
			os.Setenv(envCollectorEndpoint, opt.CollectorEndpoint)
		}
	}

	if opt.AgentEndpoint != "" {
		if set := os.Getenv(envAgentEndpoint); set == "" {
			os.Setenv(envAgentEndpoint, opt.AgentEndpoint)
		}
	}

	if opt.AgentHost != "" {
		if set := os.Getenv(envAgentHost); set == "" {
			os.Setenv(envAgentHost, opt.AgentHost)
		}
	}

	if opt.AgentPort != "" {
		if set := os.Getenv(envAgentPort); set == "" {
			os.Setenv(envAgentPort, opt.AgentPort)
		}
	}

	if opt.JaegerUser != "" {
		if set := os.Getenv(envJaegerUser); set == "" {
			os.Setenv(envJaegerUser, opt.JaegerUser)
		}
	}

	if opt.JaegerCred != "" {
		if set := os.Getenv(envJaegerCred); set == "" {
			os.Setenv(envJaegerCred, opt.JaegerCred)
		}
	}

}

type Options struct {
	Enable  bool
	Name    string
	Sampler *float64

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
