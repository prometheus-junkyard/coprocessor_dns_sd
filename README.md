# DNS based service discovery for prometheus

This daemon resolves an DNS A record per configured job, takes all returned IPs and updates prometheus.

## Configuration
The configuration consists of:
- prometheus_url: URL to the prometheus web ui

And a list of services/jobs to update from a DNS record:
- name: The name of the record to resolve.
- job-name: Optional name of the prometheus job (will take name if ommited).
- port: port the exporter is running on.
- path: path to the metrics json on the exporter.

See [sample config](dns-bridge.conf.sample)
