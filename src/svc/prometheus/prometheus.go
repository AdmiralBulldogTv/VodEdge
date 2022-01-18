package prometheus

import (
	"github.com/AdmiralBulldogTv/VodEdge/src/configure"
	"github.com/AdmiralBulldogTv/VodEdge/src/instance"

	"github.com/prometheus/client_golang/prometheus"
)

type mon struct {
	responseTimeMilliseconds prometheus.Histogram
}

func (m *mon) Register(r prometheus.Registerer) {
	r.MustRegister(
		m.responseTimeMilliseconds,
	)
}

func (m *mon) ResponseTimeMilliseconds() prometheus.Histogram {
	return m.responseTimeMilliseconds
}

func LabelsFromKeyValue(kv []configure.KeyValue) prometheus.Labels {
	mp := prometheus.Labels{}

	for _, v := range kv {
		mp[v.Key] = v.Value
	}

	return mp
}

func New(opts SetupOptions) instance.Prometheus {
	return &mon{
		responseTimeMilliseconds: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name: "edge_response_time_milliseconds",
			Help: "The response time in milliseconds",
		}),
	}
}

type SetupOptions struct {
	Labels prometheus.Labels
}
