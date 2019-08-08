package lib

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

// Node ...
type Node struct {
	Address string `mapstructure:"address"`
	Port    string `mapstructure:"port"`
	Health  string `mapstructure:"health"` // TODO
	healthy bool
}

// SetHealthy sets the node to "healthy", meaning it is perceived to be able
// to successfully process requests.
func (n *Node) SetHealthy() {
	n.healthy = true
}

// SetUnhealthy sets the node to "unhealthy", meaning it is not perceived to be
// able to successfully process requests.
func (n *Node) SetUnhealthy() {
	n.healthy = false
}

// IsHealthy returns true if healthy.
func (n *Node) IsHealthy() bool {
	return n.healthy
}

// IsUnhealthy returns true if unhealthy.
func (n *Node) IsUnhealthy() bool {
	return !n.IsHealthy()
}

// CheckHealth manually performs a health check at the node's health url. It
// returns true if healthy, false if unhealhty.
func (n *Node) CheckHealth() bool {
	url := fmt.Sprintf("http://%s:%s%s", n.Address, n.Port, n.Health)
	resp, err := http.Get(url)
	if err != nil {
		n.SetUnhealthy()
		return false
	}

	if resp.StatusCode != http.StatusOK {
		n.SetUnhealthy()
		return false
	}

	n.SetHealthy()
	return true
}

// Handler forwards request to node. Based on https://stackoverflow.com/a/34725635
func (n *Node) Handler(w http.ResponseWriter, req *http.Request) int {
	// we need to buffer the body if we want to read it here and send it
	// in the request.
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Printf("%d %s", http.StatusInternalServerError, err.Error())
		return http.StatusInternalServerError
	}

	// you can reassign the body if you need to parse it as multipart
	req.Body = ioutil.NopCloser(bytes.NewReader(body))

	// create a new url from the raw RequestURI sent by the client
	url := fmt.Sprintf("http://%s:%s/%s", n.Address, n.Port, req.RequestURI)

	proxyReq, err := http.NewRequest(req.Method, url, bytes.NewReader(body))

	// We may want to filter some headers, otherwise we could just use a shallow
	// copy proxyReq.Header = req.Header
	proxyReq.Header = make(http.Header)
	for h, val := range req.Header {
		proxyReq.Header[h] = val
	}

	client := &http.Client{}

	resp, err := client.Do(proxyReq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		log.Printf("%d %s", http.StatusBadGateway, err.Error())
		return http.StatusBadGateway
	}
	defer resp.Body.Close()

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		log.Printf("%d %s", http.StatusBadGateway, err.Error())
		return http.StatusBadGateway
	}

	if resp.StatusCode != http.StatusOK {
		w.WriteHeader(resp.StatusCode)
	}

	w.Write(b)
	return http.StatusOK
}
