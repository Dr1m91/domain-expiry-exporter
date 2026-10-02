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

	expiryDays          *prometheus.Desc
	probeSuccess        *prometheus.Desc
	lastSuccessSecods   *prometheus.Desc
	consecutiveFailures *prometheus.Desc
}

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
		consecutiveFailures: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "consecutive_failures"),
			"number of consecutive failed probe attempts since the last success",
			[]string{"domain"},
			nil,
		),
	}
}

func (c *domainCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.expiryDays
	ch <- c.probeSuccess
	ch <- c.lastSuccessSecods
	ch <- c.consecutiveFailures
}

func (c *domainCollector) Collect(ch chan<- prometheus.Metric) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	entries, err := c.store.List()
	if err != nil {
		return
	}

	for _, entry := range entries {
		ch <- prometheus.MustNewConstMetric(
			c.consecutiveFailures, prometheus.GaugeValue,
			float64(entry.ConsecutiveFailures), entry.Domain,
		)

		if !entry.HasEverSucceeded() {
			continue
		}

		ch <- prometheus.MustNewConstMetric(
			c.probeSuccess, prometheus.GaugeValue, 1, entry.Domain,
		)
		ch <- prometheus.MustNewConstMetric(
			c.expiryDays, prometheus.GaugeValue,
			mathFloorDays(entry), entry.Domain,
		)
		ch <- prometheus.MustNewConstMetric(
			c.lastSuccessSecods, prometheus.GaugeValue,
			float64(entry.LastSuccessAt.Unix()), entry.Domain,
		)
	}
}

// TODO: creates a throwaway *domainCollector on every call just to reuse
// its metric descriptors; harmless at current scale, but worth cleaning up.
func NewSingleDomainCollector(store domain.Store, target string) prometheus.Collector {
	return &singleDomainCollector{store: store, target: target}
}

type singleDomainCollector struct {
	store  domain.Store
	target string
}

func (c *singleDomainCollector) Describe(ch chan<- *prometheus.Desc) {
	base := NewDomainCollector(c.store).(*domainCollector)
	base.Describe(ch)
}

func (c *singleDomainCollector) Collect(ch chan<- prometheus.Metric) {
	entry, ok, err := c.store.Get(c.target)
	if err != nil || !ok {
		return
	}

	base := NewDomainCollector(c.store).(*domainCollector)
	ch <- prometheus.MustNewConstMetric(base.consecutiveFailures, prometheus.GaugeValue, float64(entry.ConsecutiveFailures), entry.Domain)

	if !entry.HasEverSucceeded() {
		return
	}

	ch <- prometheus.MustNewConstMetric(base.probeSuccess, prometheus.GaugeValue, 1, entry.Domain)
	ch <- prometheus.MustNewConstMetric(base.expiryDays, prometheus.GaugeValue,
		mathFloorDays(entry), entry.Domain)
	ch <- prometheus.MustNewConstMetric(base.lastSuccessSecods, prometheus.GaugeValue,
		float64(entry.LastSuccessAt.Unix()), entry.Domain)
}

func mathFloorDays(entry domain.Entry) float64 {
	return math.Floor(time.Until(entry.ExpireTime).Hours() / 24)
}
