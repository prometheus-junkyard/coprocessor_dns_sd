package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/prometheus/sd_client_golang"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var (
	configFile     = flag.String("configFile", "dns-bridge.conf", "Path to config file.")
	domainSuffix   = flag.String("domainSuffix", "dyn.example.com", "Domain suffix.")
	updateInterval = flag.Duration("updateInterval", 30*time.Second, "Update interval in seconds.")
)

type config struct {
	PrometheusUrl string `json:"prometheusUrl"`
	Services      []struct {
		Name    string `json:"name"`
		JobName string `json:"jobName"`
		Port    string `json:"port"`
		Path    string `json:"path"`
	} `json:"services"`
}

func (c *config) loadFrom(path string) (err error) {
	log.Printf("Reading config %s", path)
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		return
	}

	err = json.Unmarshal(bytes, c)
	return
}

func (c config) update(client prometheus.Client) (err error) {
	for _, service := range c.Services {
		host := service.Name + "." + *domainSuffix
		addrs, err := net.LookupIP(host)
		if err != nil {
			log.Printf("Couldn't resolve %s: %s", host, err)
			continue
		}

		var jobName string
		if service.JobName != "" {
			jobName = service.JobName
		} else {
			jobName = service.Name
		}

		targetGroup := prometheus.TargetGroup{}

		for _, addr := range addrs {
			targetGroup.Endpoints = append(targetGroup.Endpoints, fmt.Sprintf("http://%s:%s/%s", addr, service.Port, service.Path))
		}

		err = client.UpdateEndpoints(jobName, []prometheus.TargetGroup{targetGroup})
		if err != nil {
			log.Fatalf("Couldn't update Prometheus: %s", err)
		}
	}
	return
}

func (c config) handleSignals(path string) {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGHUP)

	for _ = range sig {
		err := c.loadFrom(path)
		if err != nil {
			log.Printf("Couldn't reload config: %s", err)
		}
	}
}

func main() {
	flag.Parse()

	var (
		config config
		err    error
	)

	err = config.loadFrom(*configFile)
	if err != nil {
		log.Fatalf("Couldn't read config: %s", err)
	}

	go config.handleSignals(*configFile)

	var (
		client = prometheus.New(config.PrometheusUrl)
		ticker = time.Tick(*updateInterval)
	)

	for _ = range ticker {
		log.Printf("Updating endpoints...")
		config.update(client)
	}
}
