package collector

import (
	"testing"
	"time"

	"github.com/dr1m91/domain-expiry-exporter/internal/domain"
	"github.com/prometheus/client_golang/prometheus"
)

func gather(t *testing.T, c prometheus.Collector) map[string]map[string]float64 {
	t.Helper()

	reg := prometheus.NewRegistry()
	reg.MustRegister(c)
	families, err := reg.Gather()
	if err != nil {
		t.Fatal(err)
	}

	out := make(map[string]map[string]float64)
	for _, mf := range families {
		out[mf.GetName()] = make(map[string]float64)
		for _, m := range mf.GetMetric() {
			for _, l := range m.GetLabel() {
				if l.GetName() == "domain" {
					out[mf.GetName()][l.GetValue()] = m.GetGauge().GetValue()
				}
			}
		}
	}
	return out
}

func mustSet(t *testing.T, s domain.Store, e domain.Entry) {
	t.Helper()
	if err := s.Set(e); err != nil {
		t.Fatal(err)
	}
}

func TestCollector_NewDomainExposesOnlyFailureCounter(t *testing.T) {
	store := domain.NewInMemoryStore()
	mustSet(t, store, domain.Entry{Domain: "new.com"})

	got := gather(t, NewDomainCollector(store))

	if v, ok := got["domain_consecutive_failures"]["new.com"]; !ok || v != 0 {
		t.Fatalf("expected domain_consecutive_failures=0 for new.com, got %v ok=%v", v, ok)
	}
	for _, name := range []string{"domain_expiry_days", "domain_probe_success", "domain_last_success_timestamp_seconds"} {
		if _, ok := got[name]; ok {
			t.Errorf("%s must not be exposed before the first successful probe", name)
		}
	}
}

func TestCollector_ProbedDomainExposesAllMetrics(t *testing.T) {
	store := domain.NewInMemoryStore()
	mustSet(t, store, domain.Entry{
		Domain:        "ok.com",
		ExpireTime:    time.Now().Add(10*24*time.Hour + time.Hour),
		LastSuccessAt: time.Unix(1700000000, 0),
	})

	got := gather(t, NewDomainCollector(store))

	if v := got["domain_expiry_days"]["ok.com"]; v != 10 {
		t.Errorf("domain_expiry_days = %v, want 10", v)
	}
	if v := got["domain_probe_success"]["ok.com"]; v != 1 {
		t.Errorf("domain_probe_success = %v, want 1", v)
	}
	if v := got["domain_last_success_timestamp_seconds"]["ok.com"]; v != 1700000000 {
		t.Errorf("domain_last_success_timestamp_seconds = %v, want 1700000000", v)
	}
	if v := got["domain_consecutive_failures"]["ok.com"]; v != 0 {
		t.Errorf("domain_consecutive_failures = %v, want 0", v)
	}
}

func TestCollector_FailureCounterIsExposedWithoutAnySuccess(t *testing.T) {
	store := domain.NewInMemoryStore()
	mustSet(t, store, domain.Entry{Domain: "broken.com", ConsecutiveFailures: 7})

	got := gather(t, NewDomainCollector(store))

	if v := got["domain_consecutive_failures"]["broken.com"]; v != 7 {
		t.Fatalf("domain_consecutive_failures = %v, want 7", v)
	}
}

func TestSingleDomainCollector_OnlyReportsTarget(t *testing.T) {
	store := domain.NewInMemoryStore()
	probed := domain.Entry{
		Domain:        "a.com",
		ExpireTime:    time.Now().Add(48 * time.Hour),
		LastSuccessAt: time.Unix(1700000000, 0),
	}
	mustSet(t, store, probed)
	probed.Domain = "b.com"
	mustSet(t, store, probed)

	got := gather(t, NewSingleDomainCollector(store, "a.com"))

	for name, perDomain := range got {
		if _, ok := perDomain["b.com"]; ok {
			t.Errorf("%s exposes b.com, only a.com was requested", name)
		}
	}
	if _, ok := got["domain_expiry_days"]["a.com"]; !ok {
		t.Error("expected domain_expiry_days for a.com")
	}
}

func TestSingleDomainCollector_UnknownTargetExposesNothing(t *testing.T) {
	store := domain.NewInMemoryStore()

	got := gather(t, NewSingleDomainCollector(store, "missing.com"))

	if len(got) != 0 {
		t.Fatalf("expected no metrics for an unknown target, got %v", got)
	}
}
