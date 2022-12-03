package main

import (
	"crypto/tls"
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	namespace = "mastodon"
	instanceApi = "/api/v2/instance"
	peersApi = "/api/v1/instance/peers"
	activityApi = "/api/v1/instance/activity"
)

var (
	port = flag.String("port", "9876", "the port on which to listen")
	path = flag.String("path", "/metrics", "the path on which to expose metrics")
	domain = flag.String("domain", "https://mastodon.example", "the domain on which mastodon is running")

	tr = &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client = &http.Client{Transport: tr}

	up = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "up"),
		"was the last query successful",
		nil, nil)

	// GET https://mastodon.example/api/v2/instance
	monthlyActives = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "monthly_active_users"),
		"how many users were active this month",
		nil, nil)

	// GET https://mastodon.example/api/v1/instance/peers HTTP/1.1
	numPeers = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "num_peers"),
		"the number of instances this instance is aware of",
		nil, nil)

	// GET https://mastodon.example/api/v1/instance/activity HTTP/1.1
	numStatuses = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "num_statuses"),
		"the number of statuses that have been posted",
		nil, nil)

	numLogins = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "num_logins"),
		"the number of logins the instance has seen",
		nil, nil)
)

func main() {
	instanceExporter := NewExporter(*domain)
	prometheus.MustRegister(instanceExporter)

	http.Handle(*path, promhttp.Handler())
	log.Fatal(http.ListenAndServe(":"+*port, nil))
}

type Exporter struct {
	domain string
}

func NewExporter(domain string) *Exporter {
	return &Exporter { domain: domain }
}

func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- up
	ch <- monthlyActives
	ch <- numPeers
	ch <- numStatuses
	ch <- numLogins
}

func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	actives, err := e.getInstance()
	if err != nil {
		ch <- prometheus.MustNewConstMetric(up, prometheus.GaugeValue, 0)
		log.Println(err)
		return
	}
	ch <- prometheus.MustNewConstMetric(monthlyActives, prometheus.GaugeValue, float64(actives))

	peers, err := e.getPeers()
	if err != nil {
		ch <- prometheus.MustNewConstMetric(up, prometheus.GaugeValue, 0)
		log.Println(err)
		return
	}
	ch <- prometheus.MustNewConstMetric(numPeers, prometheus.GaugeValue, float64(peers))

	statuses, logins, err := e.getActivity()
	if err != nil {
		ch <- prometheus.MustNewConstMetric(up, prometheus.GaugeValue, 0)
		log.Println(err)
		return
	}
	ch <- prometheus.MustNewConstMetric(numStatuses, prometheus.GaugeValue, float64(statuses))
	ch <- prometheus.MustNewConstMetric(numLogins, prometheus.GaugeValue, float64(logins))
	ch <- prometheus.MustNewConstMetric(up, prometheus.GaugeValue, 1)
}

func getJson(url string) (map[string]interface{}, error) {
	var m map[string]interface{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return m, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return m, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return m, err
	}

	if err := json.Unmarshal(body, &m); err != nil {
		return nil, err
	}

	return m, nil
}

func (e *Exporter) getInstance() (int, error) {
	m, err := getJson("https://"+e.domain+instanceApi)
	if err != nil {
		return 0, err
	}
	usage := m["usage"].(map[string]interface{})
	users := usage["users"].(map[string]int)
	return users["active_month"], nil
}

func (e *Exporter) getPeers() (int, error) {
	m, err := getJson("https://"+e.domain+peersApi)
	if err != nil {
		return 0, err
	}
	return len(m), nil
}

func (e *Exporter) getActivity() (statuses int, logins int, err error) {
	var m map[string]interface{}
	m, err = getJson("https://"+e.domain+activityApi)
	if err != nil {
		return 0, 0, err
	}
	week := 0
	for _, wk := range m {
		w := wk.(map[string]int)
		if w["week"] > week {
			week = w["week"]
			statuses = w["statuses"]
			logins = w["logins"]
		}
	}
	return
}
