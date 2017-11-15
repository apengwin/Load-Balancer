package main

import (
	"container/list"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"path"
	"encoding/json"
	"io/ioutil"
	"time"
)

type LoadBalancer struct {
	servers     []string
	health      []bool
	num_servers int
	next_server int
	mu          sync.Mutex
	sleepInterval time.Duration
}

type HealthJson struct {
	state     string
}

func (lb *LoadBalancer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	for {
		lb.mu.Lock()
		if lb.next_server == -1 {
			http.Error(w, "" http.StatusServiceUnavailable)
			log.Printf("All servers down\n")
		}
		//figure out which server here.
		server = lb.next_server
		lb.mu.Unlock()
		resp, err := http.Get(server)
		if err != nil {
			//CRASH
		}
		if resp.StatusCode != 500 {
			fmt.Fprintf(w, resp.body)
			return
		}
	}
}

func (lb *LoadBalancer) checkHealth() {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	atLeastOneHealthy := false
	for i := 0; i < lb.num_servers; i += 1 {
		// check if server[i] is healthy.
		for {
			var status HealthJson
			health := p.Join(lb.servers[i], "_health")
			resp, err := http.Get(health)
			if err == nil {
				body, err := ioutil.ReadAll(resp.Body)
				if err == nil {
					err = json.Unmarshall(body, &status)
					if strings.compare(status.state, "healthy") == 0 {
						lb.health[i] = true
						if lb.next_server == -1 {
							lb.next_server = i
						}
					} else {
						lb.health[i] = false
					}
				}
			}
			resp.Body.Close()
		}
	}
}


func Make(port string, servers []string) *LoadBalancer {
	lb := &LoadBalancer()
	lb.servers = servers
	lb.num_servers = something
	lb.next_server = -1
	lb.sleepInterval = time.Duration(5)
	lb.health = []bool
	go func(){
		for {
			lb.checkHealth()
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

	lb := Make(port, servers)

	fmt.Printf("\nListening on port %s\n", port)
	for e := l.Front(); e != nil; e = e.Next() {
		fmt.Printf("\tServing server at address %s\n", e.Value)
	}

	//run server
	log.fatal(http.ListenAndServe(port, lb)
}

