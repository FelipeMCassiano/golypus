package loadbalancer

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync/atomic"
)

// TODO:
// after a container be scaled use the loadbalance to proxy all reqs

type LoadBalancer struct {
	Port        string
	ServerPorts ContainerPorts
}

func (l *LoadBalancer) Serve() error {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		s := l.Choose()

		u, err := url.Parse(s)
		if err != nil {
			http.Error(w, "failed to parse server URL", http.StatusBadGateway)
			return
		}

		httputil.NewSingleHostReverseProxy(u).ServeHTTP(w, r)
	})

	return http.ListenAndServe(fmt.Sprintf(":%s", l.Port), nil)
}

func (l *LoadBalancer) Choose() string {
	var serverCount int32 = 0
	for {
		current := atomic.LoadInt32(&serverCount)
		next := current + 1
		if next >= int32(len(l.ServerPorts)) {
			next = 0
		}

		if atomic.CompareAndSwapInt32(&serverCount, current, next) {
			return l.ServerPorts[current]
		}
	}
}

func NewLoadBalancer(port string, serverPorts ContainerPorts) *LoadBalancer {
	return &LoadBalancer{Port: port, ServerPorts: serverPorts}
}

type ContainerPorts []string
