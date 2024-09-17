package loadbalancer

// TODO:
// after a container be scaled use the loadbalance to proxy all reqs

type LoadBalancer struct {
	Port        string
	ServerPorts []ContainerPorts
}

func (l *LoadBalancer) Serve() {
}

func (l *LoadBalancer) Choose() {
}

func NewLoadBalancer(port string, serverPorts []ContainerPorts) *LoadBalancer {
	return &LoadBalancer{Port: port, ServerPorts: serverPorts}
}

type ContainerPorts map[string]string
