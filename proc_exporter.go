package main

import (
	"flag"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/log"
	"github.com/prometheus/common/version"

	"github.com/catawiki/proc_exporter/collector"
)

func main() {
	var (
		procfsPath    = flag.String("procfs", "/proc", "path to read proc data from")
		configPath    = flag.String("config.path", "", "path to YAML config file")
		metricsPath   = flag.String("web.telemetry-path", "/metrics", "Path under which to expose metrics.")
		listenAddress = flag.String("web.listen-address", ":9256", "Address to listen on for web interface and telemetry.")
	)
	flag.Parse()

	log.Infoln("Starting proc_exporter", version.Info())
	log.Infoln("Build context", version.BuildContext())

	var matchnamer collector.MatchNamer

	if *configPath != "" {
		cfg, err := collector.ReadConfig(*configPath)
		if err != nil {
			log.Fatalf("Error reading config file %q: %v", *configPath, err)
		}
		log.Infoln("Reading metrics from %s based on %q", *procfsPath, *configPath)
		matchnamer = cfg.MatchNamers
	}

	prometheus.MustRegister(collector.NewProcCollector(*procfsPath, matchnamer))

	http.Handle(*metricsPath, promhttp.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
			<head><title>Proc Exporter</title></head>
			<body>
			<h1>Proc Exporter</h1>
			<p><a href="` + *metricsPath + `">Metrics</a></p>
			</body>
			</html>`))
	})

	log.Infoln("Listening on", *listenAddress)
	log.Fatal(http.ListenAndServe(*listenAddress, nil))
}
