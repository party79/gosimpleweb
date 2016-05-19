package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
)

type proxyClientJob struct {
	process func(proxy *httputil.ReverseProxy)
}

func (job *proxyClientJob) Run(proxy *httputil.ReverseProxy) {
	job.process(proxy)
}

type proxyClients struct {
	name     string
	servers  cfgServerList
	workerCh chan *proxyClientJob
	closing  chan bool
}

func (proxy *proxyClients) Start(client *httputil.ReverseProxy, id int) {
	for {
		select {
		case job := <-proxy.workerCh:
			go func() {
				job.Run(client)
			}()
		case <-proxy.closing:
			return
		}
	}
}

func (proxy *proxyClients) Init() {
	proxy.servers.Each(func(idx int, server string) bool {
		if serverUrl, err := url.Parse(server); err == nil {
			client := httputil.NewSingleHostReverseProxy(serverUrl)
			log.Printf("Starting FCgi: name=%s server=%s", proxy.name, server)
			go proxy.Start(client, idx)
		}
		return true
	})
}
func (proxy *proxyClients) Kill() {
	for _, _ = range proxy.servers {
		proxy.closing <- true
	}
}

func (proxy *proxyClients) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	respch := make(chan *httputil.ReverseProxy, 1)
	proxy.workerCh <- &proxyClientJob{
		process: func(proxy *httputil.ReverseProxy) {
			respch <- proxy
		},
	}
	client := <-respch
	client.ServeHTTP(rw, req)
}

func newProxyClient(name string, servers cfgServerList) (proxy *proxyClients) {
	proxy = &proxyClients{
		name:     name,
		servers:  servers,
		workerCh: make(chan *proxyClientJob, len(servers)*100),
		closing:  make(chan bool, len(servers)),
	}
	proxy.Init()
	return
}
