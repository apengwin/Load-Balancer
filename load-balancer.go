package main

import (
	"bytes"
	"container/list"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

//
// A load balancer instance.
// The load balancer chooses among healthy server instances
// in a round-robin fashion.
// 
type LoadBalancer struct {
	servers       []*url.URL
	health        []bool
	last_used     int
	mu            sync.Mutex
	sleepInterval time.Duration
}

//
// Utility method used to copy headers from one request to another.
//
func copyHeader(src, dest http.Header) {
	//headers are stored as a map [string][]string
	for headerName, vals := range src {
		for _, v := range vals {
			dest.Add(headerName, v)
		}
	}
}

//
// Handler for an HTTP session. Loop until we successfully process a request.
//
func (lb *LoadBalancer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	for {
		lb.mu.Lock()
		// Use last_server == -1 as a flag that all servers are down.
		if lb.last_used == -1 {
			http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
			log.Printf("All servers down\n")
			lb.mu.Unlock()
			return
		}
		server := lb.last_used
		for !lb.health[server] {
			server = (server + 1) % len(lb.servers)
		}
		lb.last_used = server
		lb.mu.Unlock()

		// Create a new request for th
		proxyURL := lb.servers[server].String()
		proxyBody, _ := ioutil.ReadAll(r.Body)

		proxyReq, err := http.NewRequest(r.Method, proxyURL, bytes.NewReader(proxyBody))

		if err != nil {
			log.Fatal(err)
		}

		resp, err := http.DefaultClient.Do(proxyReq)
		if err != nil {
			log.Fatal(err)
		}

		// if status 500, try another server.
		if resp.StatusCode != 500 {
			copyHeader(resp.Header, w.Header())
			io.Copy(w, resp.Body)
			w.WriteHeader(resp.StatusCode)
			return
		}
	}
}


//
// Poll all of the servers to see which ones are healthy.
// Update lb.health array
// This function loops in the background while lb handles connections. Check Make()
//
func (lb *LoadBalancer) updateHealth() {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	atLeastOneHealthy := false
	for i := 0; i < len(lb.servers); i += 1 {
		// check if server[i] is healthy.
		//eventualy this should be threaded
		if lb.checkHealth(i) {
			atLeastOneHealthy = true
		}
	}
	if !atLeastOneHealthy {
		lb.last_used = -1
	}
}

//
// Poll the status for a given server.
//
func (lb *LoadBalancer) checkHealth(server int) bool {

	var status map[string]interface{}

	health, err := lb.servers[server].Parse("_health")
	if err != nil {
		log.Fatal(err)
	}

	//loop until successful.
	for {
		resp, err := http.Get(health.String())
		defer resp.Body.Close()
		if err != nil {
			log.Println(err)
		}
		if err != nil {
			log.Println(err)
		}
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}
		err = json.Unmarshal(body, &status)
		if err != nil {
			log.Fatal(err)
		}
		if strings.Compare(status["state"].(string), "healthy") == 0 {
			lb.health[server] = true
			if lb.last_used == -1 {
				lb.last_used = server
			}
		} else {
			lb.health[server] = false
		}
		return lb.health[server]
	}
}


// Set up everything and return a load balancer
func Make(servers *list.List) *LoadBalancer {

	lb := &LoadBalancer{}
	lb.servers = make([]*url.URL, servers.Len())
	i := 0
	for e := servers.Front(); e != nil; e = e.Next() {
		url, err := url.Parse(e.Value.(string))
		if err != nil {
			log.Fatal(err)
		}
		lb.servers[i] = url
		i += 1
	}

	lb.last_used = -1
	lb.sleepInterval = time.Duration(5)
	lb.health = make([]bool, servers.Len())

	// Start polling server statuses.
	go func() {
		for {
			lb.updateHealth()
			time.Sleep(time.Millisecond * lb.sleepInterval)
		}
	}()
	return lb
}

func main() {

	fmt.Printf("starting load balancer...\n")

	servers := list.New()
	var port string
	var readPort, readServer bool = false, false

	// Read command line inputs.
	for _, str := range os.Args {
		if strings.Compare(str, "-b") == 0 {
			readServer = true
			readPort = false
		} else if strings.Compare(str, "-p") == 0 {
			readPort = true
			readServer = false
		} else {
			if readServer {
				servers.PushBack(str)
			}
			if readPort {
				port = str
			}
			readServer = false
			readPort = false
		}
	}

	lb := Make(servers)
	fmt.Printf("\nListening on port %s\n", port)
	for e := servers.Front(); e != nil; e = e.Next() {
		fmt.Printf("\tServing server at address %s\n", e.Value)
	}

	//run server
	log.Fatal(http.ListenAndServe(":"+port, lb))
}
