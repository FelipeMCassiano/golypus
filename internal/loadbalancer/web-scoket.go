package loadbalancer

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/gorilla/websocket"
	"golang.org/x/sync/errgroup"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type CreateLoadbalancerReq struct {
	LoadBalancerPort string         `json:"loadBalancerPort"`
	Ports            ContainerPorts `json:"ports"`
}

func InitWSLoadBalancer() {
	http.HandleFunc("/loadbalancer/create", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		defer conn.Close()

		lbChan := make(chan LoadBalancer, 1)

		group, _ := errgroup.WithContext(context.Background())

		group.Go(func() error {
			for {
				_, message, err := conn.ReadMessage()
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return err
				}

				if err := handlerWsMessage(message, lbChan); err != nil {
					return err
				}

			}
		})

		group.Go(func() error {
			for lb := range lbChan {
				if err := lb.Serve(); err != nil {
					return err
				}
			}
			return nil
		})

		if err := group.Wait(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})

	http.ListenAndServe(":4444", nil)
}

func handlerWsMessage(message []byte, lbChan chan LoadBalancer) error {
	var loadbalancerReq CreateLoadbalancerReq

	if err := json.Unmarshal(message, &loadbalancerReq); err != nil {
		return err
	}

	lbChan <- *NewLoadBalancer(loadbalancerReq.LoadBalancerPort, loadbalancerReq.Ports)
	return nil
}
