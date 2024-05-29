package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	//	"os/signal"
	"net/http"
	//	"strings"
	//	"time"

	"github.com/slashdoom/aruba_exporter/config"
	"github.com/slashdoom/aruba_exporter/connector"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	log "github.com/sirupsen/logrus"
)

const version string = "0.0.1"

var (
	showVersion   = flag.Bool("version", false, "Print version information.")
	listenAddress = flag.String("web.listen-address", ":9909", "Address on which to expose metrics and web interface.")
	metricsPath   = flag.String("web.telemetry-path", "/metrics", "Path under which to expose metrics.")
	sshHosts      = flag.String("ssh.targets", "", "Hosts to scrape")
	sshUsername   = flag.String("ssh.user", "aruba_exporter", "Username to use when connecting to devices using ssh")
	sshKeyFile    = flag.String("ssh.keyfile", "", "Public key file to use when connecting to devices using ssh")
	sshPassword   = flag.String("ssh.password", "", "Password to use when connecting to devices using ssh")
	sshTimeout    = flag.Int("ssh.timeout", 5, "Timeout to use for SSH connection")
	sshBatchSize  = flag.Int("ssh.batch-size", 10000, "The SSH response batch size")
	level         = flag.String("level", "info", "Set logging verbose level")
	configFile    = flag.String("config.file", "", "Path to config file")
	devices       []*connector.Device
	cfg           *config.Config
)

func init() {
	log.SetOutput(os.Stdout)
	flag.Usage = func() {
		fmt.Println("Usage: aruba_exporter [ ... ]\n\nParameters:")
		fmt.Println()
		flag.PrintDefaults()
	}
}

func main() {
	flag.Parse()

	if *showVersion {
		printVersion()
		os.Exit(0)
	}

	err := initialize()
	if err != nil {
		log.Fatalf("could not initialize exporter. %v", err)
	}

	startServer()
}

func initialize() error {
	c, err := loadConfig()
	if err != nil {
		return err
	}
	l, err := log.ParseLevel(c.Level)
	if err == nil {
		log.SetLevel(l)
	}

	devices, err = devicesForConfig(c)
	if err != nil {
		return err
	}
	cfg = c

	return nil
}

func printVersion() {
	fmt.Println("aruba_exporter")
	fmt.Printf("Version: %s\n", version)
	fmt.Println("Author(s): slashdoom (Patrick Ryon)")
	fmt.Println("Metric exporter for Aruba switches, controllers and instant APs")
}

func loadConfig() (*config.Config, error) {
	l, err := log.ParseLevel(*level)
	if err == nil {
		log.SetLevel(l)
	}

	if len(*configFile) == 0 {
		log.Infoln("Loading config flags")
		return loadConfigFromFlags(), nil
	}

	log.Infoln("Loading config from", *configFile)
	b, err := ioutil.ReadFile(*configFile)
	if err != nil {
		return nil, err
	}

	c, err := config.Load(bytes.NewReader(b))
	if c.Password == "" {
		log.Debugln("Loaded password from env")
		c.Password = os.Getenv("SSH_PASSWORD")
	}
}

func loadConfigFromFlags() *config.Config {
	c := config.New()

	c.Level = *level
	c.Timeout = *sshTimeout
	c.BatchSize = *sshBatchSize
	c.Username = *sshUsername
	c.Password = *sshPassword
	c.KeyFile = *sshKeyFile
	c.DevicesFromTargets(*sshHosts)
	log.Debugln(c)

	f := c.Features
	log.Debugln(f)

	return c
}

func startServer() {
	log.Infof("starting aruba_exporter (version: %s)\n", version)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`
<html>
  <head>
    <title>Aruba Exporter (Version ` + version + `)</title>
  </head>
  <body>
    <h1>Aruba Exporter</h1>
    <p><a href="` + *metricsPath + `">Metrics</a></p>
    <h2>More information:</h2>
    <p><a href="https://github.com/slashdoom/aruba_exporter">github.com/slashdoom/aruba_exporter</a></p>
  </body>
</html>
`))
	})
	http.HandleFunc(*metricsPath, handleMetricsRequest)

	log.Infof("Listening for %s on %s\n", *metricsPath, *listenAddress)
	log.Fatal(http.ListenAndServe(*listenAddress, nil))
}

func handleMetricsRequest(w http.ResponseWriter, r *http.Request) {
	reg := prometheus.NewRegistry()

	a := newArubaCollector(devices)
	reg.MustRegister(a)

	promhttp.HandlerFor(reg, promhttp.HandlerOpts{
		ErrorLog:      log.New(),
		ErrorHandling: promhttp.ContinueOnError}).ServeHTTP(w, r)
}
