package loadbalancer

// TODO:
// after a container be scaled use the loadbalance to proxy all reqs

type LoadBalancer struct {
	Servers []*Server
	Port    string
}

func (l *LoadBalancer) Serve() {
}

func (l *LoadBalancer) Choose() {
}

func NewLoadBalancer() *LoadBalancer {
	return &LoadBalancer{}
}

type Server struct {
	Port string
}
