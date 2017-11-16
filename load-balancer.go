package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"
	"time"
	"container/list"
)

type LoadBalancer struct {
	servers       []string
	health        []bool
	last_used     int
	mu            sync.Mutex
	sleepInterval time.Duration
}

type HealthJson struct {
	state string
}

func copyHeader(src, dest http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dest.Add(k, v)
		}
	}
}

func (lb *LoadBalancer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	for {
		lb.mu.Lock()
		if lb.last_used == -1 {
			http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
			log.Printf("All servers down\n")
			lb.mu.Unlock()
			return
		}
		//figure out which server here.
		server := lb.last_used
		for !lb.health[server] {
			server = (server + 1) % len(lb.servers)
		}
		lb.last_used = server
		lb.mu.Unlock()
		req.URL = lb.servers[server]
		resp, _ := http.DefaultClient.Do(req)
		if resp.StatusCode != 500 {
			copyHeader(resp.header, w.Header())
			w.writeHeader(resp.StatusCode)
			io.Copy(w, resp.Body)
			return
		}
	}
}

func (lb *LoadBalancer) checkHealth(server int) {
	var status HealthJson
	health := path.Join(lb.servers[server], "_health")
	for {
		resp, err := http.DefaultClient.Do(health)
		defer resp.Body.Close()
		if err == nil {
			body, err := ioutil.ReadAll(resp.Body)
			if err == nil {
				err = json.Unmarshal(body, &status)
				if strings.Compare(status.state, "healthy") == 0 {
					lb.health[server] = true
					if lb.last_used == -1 {
						lb.last_used = server
					}
				} else {
					lb.health[server] = false
				}
				return
			}
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
		lb.checkHealth(i)
	}
}


func Make(servers *list.List) *LoadBalancer {

	lb := &LoadBalancer{}
	lb.servers = make([]string, servers.Len())
	lb.last_used = -1
	lb.sleepInterval = time.Duration(5)
	lb.health = make([]bool, servers.Len())
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
	fmt.Println(port)
	fmt.Println(servers)
	fmt.Printf("\nListening on port %s\n", port)
	for e := servers.Front(); e != nil; e = e.Next() {
		fmt.Printf("\tServing server at address %s\n", e.Value)
	}

	//run server
	log.Fatal(http.ListenAndServe(":"+port, lb))
}
