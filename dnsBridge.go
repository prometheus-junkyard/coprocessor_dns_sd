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
	updateInterval = flag.Int("updateInterval", 30, "Update interval in seconds.")
	conf           config
	sig            chan os.Signal
)

type config struct {
	PrometheusUrl string `json:"prometheus_url"`
	Services      []struct {
		Name    string `json:"name"`
		JobName string `json:"job-name"`
		Port    string `json:"port"`
		Path    string `json:"path"`
	} `json:"services"`
}

func init() {
	flag.Parse()
	sig = make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGHUP)
	go func() {
		for _ = range sig {
			err := readConfig()
			if err != nil {
				log.Printf("Couldn't reload config: %s", err)
			}
		}
	}()
}

func readConfig() (err error) {
	log.Printf("reading config %s", *configFile)
	bytes, err := ioutil.ReadFile(*configFile)
	if err != nil {
		return
	}

	err = json.Unmarshal(bytes, &conf)
	return
}

func update(client prometheus.Client) (err error) {
	for _, service := range conf.Services {
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
			log.Fatalf("Couldn't update prometheus: %s", err)
		}
	}
	return
}

func main() {
	err := readConfig()
	if err != nil {
		log.Fatalf("Couldn't read config: %s", err)
	}

	client := prometheus.New(conf.PrometheusUrl)
	ticker := time.Tick(time.Duration(*updateInterval) * time.Second)
	update(client)
	for _ = range ticker {
		log.Printf("updating endpoints.")
		update(client)
	}
}
