package main

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	namespace     = "mastodon"
	instanceV1Api = "/api/v1/instance"
	instanceV2Api = "/api/v2/instance"
	peersApi      = "/api/v1/instance/peers"
	activityApi   = "/api/v1/instance/activity"
)

var (
	port   = flag.String("port", "9876", "the port on which to listen")
	path   = flag.String("path", "/metrics", "the path on which to expose metrics")
	domain = flag.String("domain", "https://mastodon.example", "the domain on which mastodon is running")

	tr = &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client = &http.Client{Transport: tr}

	up = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "up"),
		"was the last query successful",
		nil, nil)

	// GET https://mastodon.example/api/v1/instance
	userCount = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "user_count"),
		"number of users", nil, nil)

	statusCount = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "status_count"),
		"number of statuses", nil, nil)

	domainCount = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "domain_count"),
		"number of domains", nil, nil)

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
		"the number of statuses that have been posted in the given week",
		[]string{"week"}, nil)

	numLogins = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "num_logins"),
		"the number of logins the instance has seen in the given week",
		[]string{"week"}, nil)
)

func main() {
	flag.Parse()

	instanceExporter := NewExporter(*domain)
	prometheus.MustRegister(instanceExporter)

	http.Handle(*path, promhttp.Handler())

	log.Printf("exporting from %s", *domain)
	log.Printf("listening on :%s%s", *port, *path)

	log.Fatal(http.ListenAndServe(":"+*port, nil))
}

type Exporter struct {
	domain string
}

func NewExporter(domain string) *Exporter {
	return &Exporter{domain: domain}
}

func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- up
	ch <- userCount
	ch <- statusCount
	ch <- domainCount
	ch <- monthlyActives
	ch <- numPeers
	ch <- numStatuses
	ch <- numLogins
}

func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	users, statuses, domains, err := e.getInstanceV1()
	if err != nil {
		ch <- prometheus.MustNewConstMetric(up, prometheus.GaugeValue, 0)
		log.Println(err)
		return
	}
	ch <- prometheus.MustNewConstMetric(userCount, prometheus.GaugeValue, users)
	ch <- prometheus.MustNewConstMetric(statusCount, prometheus.GaugeValue, statuses)
	ch <- prometheus.MustNewConstMetric(domainCount, prometheus.GaugeValue, domains)

	actives, err := e.getInstanceV2()
	if err != nil {
		ch <- prometheus.MustNewConstMetric(up, prometheus.GaugeValue, 0)
		log.Println(err)
		return
	}
	ch <- prometheus.MustNewConstMetric(monthlyActives, prometheus.GaugeValue, actives)

	peers, err := e.getPeers()
	if err != nil {
		ch <- prometheus.MustNewConstMetric(up, prometheus.GaugeValue, 0)
		log.Println(err)
		return
	}
	ch <- prometheus.MustNewConstMetric(numPeers, prometheus.GaugeValue, float64(peers))

	statusesPerWeek, loginsPerWeek, err := e.getActivity()
	if err != nil {
		ch <- prometheus.MustNewConstMetric(up, prometheus.GaugeValue, 0)
		log.Println(err)
		return
	}
	for week, statuses := range statusesPerWeek {
		ch <- prometheus.MustNewConstMetric(numStatuses, prometheus.GaugeValue, float64(statuses), week)
	}
	for week, logins := range loginsPerWeek {
		ch <- prometheus.MustNewConstMetric(numLogins, prometheus.GaugeValue, float64(logins), week)
	}
	ch <- prometheus.MustNewConstMetric(up, prometheus.GaugeValue, 1)
	log.Println("collected metrics successfully")
}

func getJsonMap(url string) (map[string]interface{}, error) {
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

func getJsonArray(url string) ([]interface{}, error) {
	var a []interface{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return a, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return a, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return a, err
	}

	if err := json.Unmarshal(body, &a); err != nil {
		return nil, err
	}

	return a, nil
}

func (e *Exporter) getInstanceV1() (float64, float64, float64, error) {
	m, err := getJsonMap("https://" + e.domain + instanceV1Api)
	if err != nil {
		return 0, 0, 0, err
	}
	stats, ok := m["stats"].(map[string]interface{})
	if !ok {
		return 0, 0, 0, err
	}

	return stats["user_count"].(float64), stats["status_count"].(float64), stats["domain_count"].(float64), nil
}

func (e *Exporter) getInstanceV2() (float64, error) {
	m, err := getJsonMap("https://" + e.domain + instanceV2Api)
	if err != nil {
		return 0, err
	}
	usage, ok := m["usage"].(map[string]interface{})
	if !ok {
		return 0, errors.New("unable to parse 'usage' from instance API")
	}
	users, ok := usage["users"].(map[string]interface{})
	if !ok {
		return 0, errors.New("unable to parse 'users' from instance API usage")
	}
	return users["active_month"].(float64), nil
}

func (e *Exporter) getPeers() (int, error) {
	m, err := getJsonArray("https://" + e.domain + peersApi)
	if err != nil {
		return 0, err
	}
	return len(m), nil
}

func (e *Exporter) getActivity() (statusesPerWeek map[string]int, loginsPerWeek map[string]int, err error) {
	var a []interface{}
	a, err = getJsonArray("https://" + e.domain + activityApi)
	if err != nil {
		return
	}
	statusesPerWeek = make(map[string]int)
	loginsPerWeek = make(map[string]int)
	for _, wk := range a {
		w := wk.(map[string]interface{})
		week := w["week"].(string)
		statusesPerWeek[week], _ = strconv.Atoi(w["statuses"].(string))
		loginsPerWeek[week], _ = strconv.Atoi(w["logins"].(string))
	}
	return
}
