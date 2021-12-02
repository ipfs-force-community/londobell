package tracing

import (
	"strings"

	"go.opencensus.io/trace"
)

type logicOperator int

const (
	logicAnd logicOperator = iota
	logicOr
)

var defaultKeys = []string{"api.call"}

func FilterSample(keys []string) trace.Sampler {
	return func(sp trace.SamplingParameters) trace.SamplingDecision {
		for _, v := range keys {
			if strings.Contains(sp.Name, v) {
				return trace.SamplingDecision{Sample: false}
			}
		}
		return trace.SamplingDecision{Sample: true}
	}
}

func MixinSample(operator logicOperator, samplers ...trace.Sampler) trace.Sampler {
	return func(sp trace.SamplingParameters) trace.SamplingDecision {
		var sampleFlag = true
		switch operator {
		case logicAnd:
			for _, sampler := range samplers {
				sampleFlag = sampleFlag && sampler(sp).Sample
			}
		case logicOr:
			sampleFlag = false
			for _, sampler := range samplers {
				sampleFlag = sampleFlag || sampler(sp).Sample
			}
		}
		return trace.SamplingDecision{Sample: sampleFlag}
	}
}
