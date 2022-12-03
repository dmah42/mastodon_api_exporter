# mastodon_api_exporter
a prometheus exporter that gets some metrics from the mastodon api

while it's possible to get metrics into prometheus from mastodon using the
[statsd exporter](https://dev.to/kirklewis/metrics-with-prometheus-statsd-exporter-and-grafana-5145),
the set of metrics that are exposed does not include those that are exposed in
[the mastodon API](https://docs.joinmastodon.org/methods/instance/).

this exporter gets some metrics from that API into prometheus.

## usage

```
./mastodon_api_exporter --help
Usage of ./mastodon_api_exporter:
  -domain string
    	the domain on which mastodon is running (default "https://mastodon.example")
  -path string
    	the path on which to expose metrics (default "/metrics")
  -port string
    	the port on which to listen (default "9876")
```

## exported metrics

for example:

```
# HELP mastodon_monthly_active_users how many users were active this month
# TYPE mastodon_monthly_active_users gauge
mastodon_monthly_active_users 354
# HELP mastodon_num_logins the number of logins the instance has seen in the given week
# TYPE mastodon_num_logins gauge
mastodon_num_logins{week="1663363022"} 0
mastodon_num_logins{week="1663967822"} 0
mastodon_num_logins{week="1664572622"} 0
mastodon_num_logins{week="1665177422"} 0
mastodon_num_logins{week="1665782222"} 0
mastodon_num_logins{week="1666387022"} 0
mastodon_num_logins{week="1666991822"} 0
mastodon_num_logins{week="1667596622"} 0
mastodon_num_logins{week="1668201422"} 0
mastodon_num_logins{week="1668806222"} 0
mastodon_num_logins{week="1669411022"} 0
mastodon_num_logins{week="1670015822"} 4000
# HELP mastodon_num_peers the number of instances this instance is aware of
# TYPE mastodon_num_peers gauge
mastodon_num_peers 12000
# HELP mastodon_num_statuses the number of statuses that have been posted in the given week
# TYPE mastodon_num_statuses gauge
mastodon_num_statuses{week="1663363022"} 0
mastodon_num_statuses{week="1663967822"} 0
mastodon_num_statuses{week="1664572622"} 0
mastodon_num_statuses{week="1665177422"} 0
mastodon_num_statuses{week="1665782222"} 0
mastodon_num_statuses{week="1666387022"} 0
mastodon_num_statuses{week="1666991822"} 0
mastodon_num_statuses{week="1667596622"} 0
mastodon_num_statuses{week="1668201422"} 0
mastodon_num_statuses{week="1668806222"} 0
mastodon_num_statuses{week="1669411022"} 0
mastodon_num_statuses{week="1670015822"} 10000
# HELP mastodon_up was the last query successful
# TYPE mastodon_up gauge
mastodon_up 1
```
