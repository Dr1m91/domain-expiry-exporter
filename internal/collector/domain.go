package collector

import (
	"math"
	"sync"
	"time"

	"github.com/dr1m91/domain-expiry-exporter/internal/domain"
	"github.com/prometheus/client_golang/prometheus"
)

type domainCollector struct {
	mutex sync.Mutex
	store domain.Store

	expiryDays        *prometheus.Desc
	probeSuccess      *prometheus.Desc
	lastSuccessSecods *prometheus.Desc
}

// NewDomainCollector returns a collector that reads domain state from store.
func NewDomainCollector(store domain.Store) prometheus.Collector {
	const namespace = "domain"
	return &domainCollector{
		store: store,
		expiryDays: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "expiry_days"),
			"time in days until the domain expires",
			[]string{"domain"},
			nil,
		),
		probeSuccess: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "probe_success"),
			"whether we have ever successfully probed this domain",
			[]string{"domain"},
			nil,
		),
		lastSuccessSecods: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "last_success_timestamp_seconds"),
			"unix timestamp of the last successful probe",
			[]string{"domain"},
			nil,
		),
	}
}

func (c *domainCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.expiryDays
	ch <- c.probeSuccess
	ch <- c.lastSuccessSecods
}

func (c *domainCollector) Collect(ch chan<- prometheus.Metric) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	entries, err := c.store.List()
	if err != nil {
		return
	}

	for _, entry := range entries {
		if !entry.HasEverSucceeded() {
			continue
		}

		ch <- prometheus.MustNewConstMetric(
			c.probeSuccess, prometheus.GaugeValue, 1, entry.Domain,
		)
		ch <- prometheus.MustNewConstMetric(
			c.expiryDays, prometheus.GaugeValue,
			math.Floor(time.Until(entry.ExpireTime).Hours()/24), entry.Domain,
		)
		ch <- prometheus.MustNewConstMetric(
			c.lastSuccessSecods, prometheus.GaugeValue,
			float64(entry.LastSuccessAt.Unix()), entry.Domain,
		)
	}
}
