package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/alecthomas/kingpin/v2"
	"github.com/dr1m91/domain-expiry-exporter/internal/collector"
	"github.com/dr1m91/domain-expiry-exporter/internal/domain"
	"github.com/dr1m91/domain-expiry-exporter/internal/probe"
	"github.com/dr1m91/domain-expiry-exporter/internal/safeconfig"
	"github.com/dr1m91/domain-expiry-exporter/internal/scheduler"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// nolint: gochecknoglobals
var (
	bind         = kingpin.Flag("bind", "addr to bind the server").Short('b').Default(":9222").String()
	debug        = kingpin.Flag("debug", "show debug logs").Default("false").Bool()
	format       = kingpin.Flag("logFormat", "log format to use").Default("console").Enum("json", "console")
	concurrency  = kingpin.Flag("concurrency", "max concurrent RDAP/whois checks").Default("20").Int()
	scanInterval = kingpin.Flag("scan-interval", "how often the scheduler scans for due domains").Default("1m").Duration()
	configFile   = kingpin.Flag("config", "optional static list of domains to seed (for setups without vmagent /probe scraping)").String()
	version      = "dev"
)

func main() {
	kingpin.Version("domain-expiry-exporter version " + version)
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()

	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if *format == "console" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}
	if *debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
		log.Debug().Msg("enabled debug mode")
	}

	log.Info().Msgf("starting domain-expiry-exporter %s", version)

	store := domain.NewInMemoryStore()
	client := probe.NewMultiClient(probe.NewRDAPClient(), probe.NewWhoisClient())

	if *configFile != "" {
		cfg, err := safeconfig.New(*configFile)
		if err != nil {
			log.Fatal().Err(err).Msg("error to create config")
		}
		for _, d := range cfg.Domains {
			if _, ok, _ := store.Get(d.Name); !ok {
				if err := store.Set(domain.Entry{Domain: d.Name}); err != nil {
					log.Error().Err(err).Msgf("failed to seed %s from config", d.Name)
				}
			}
		}
		log.Info().Msgf("seeded %d domains from config file", len(cfg.Domains))
	}

	sched := &scheduler.Scheduler{
		Store:        store,
		Client:       client,
		Concurrency:  *concurrency,
		ScanInterval: *scanInterval,
	}

	wg := &sync.WaitGroup{}
	defer wg.Wait()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wg.Add(1)
	go func() {
		defer wg.Done()
		sched.Run(ctx)
	}()

	prometheus.DefaultRegisterer.MustRegister(collector.NewDomainCollector(store))

	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/probe", probeHandler(store))
	http.HandleFunc("/debug/store", debugStoreHandler(store))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, `
			<html>
			<head><title>Domain Exporter</title></head>
			<body>
				<h1>Domain Exporter</h1>
				<p><a href="/metrics">Metrics</a></p>
			</body>
			</html>
			`,
		)
	})

	if err := runServerWithGracefullyShutdown(wg); err != nil {
		log.Fatal().Err(err).Msg("error starting server")
	}

	log.Info().Msg("domain exporter is finished")
}

func runServerWithGracefullyShutdown(wg *sync.WaitGroup) error {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGTERM)
	signal.Notify(signalChan, syscall.SIGINT)

	server := &http.Server{Addr: *bind}

	wg.Add(1)
	go func() {
		defer wg.Done()
		sig := <-signalChan

		log.Warn().Msgf("got %s signal. Shutdown", sig)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			log.Error().Err(err).Msg("failed to shutdown http server")
		}
	}()

	log.Info().Msgf("listening on %s", *bind)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}

	return nil
}

func probeHandler(store domain.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		target := strings.TrimPrefix(r.URL.Query().Get("target"), "www.")
		if target == "" {
			http.Error(w, "target parameter is missing", http.StatusBadRequest)
			return
		}

		entry, ok, err := store.Get(target)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		now := time.Now()
		if !ok {
			entry = domain.Entry{Domain: target}
		}
		entry.LastRequestedAt = now
		if err := store.Set(entry); err != nil {
			log.Error().Err(err).Msgf("failed to record heartbeat for %s", target)
		}

		registry := prometheus.NewRegistry()
		registry.MustRegister(collector.NewSingleDomainCollector(store, target))
		promhttp.HandlerFor(registry, promhttp.HandlerOpts{}).ServeHTTP(w, r)
	}
}

// TODO: this endpoint dumps the entire registry unauthenticated; fine for
// local development, but should be removed or gated behind a flag
// (e.g. --enable-debug-endpoints) before staging/production.
func debugStoreHandler(store domain.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		entries, err := store.List()
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		_ = enc.Encode(entries)
	}
}
