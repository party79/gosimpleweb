package main

import (
	"log"

	"github.com/tomasen/fcgi_client"
)

type fCgiClientJob struct {
	process func(fcgi *fcgiclient.FCGIClient, err error)
}

func (job *fCgiClientJob) Run(fcgi *fcgiclient.FCGIClient, err error) {
	job.process(fcgi, err)
}

type fCgiClients struct {
	name     string
	servers  cfgServerList
	workerCh chan *fCgiClientJob
	closing  chan bool
}

func (fcgi *fCgiClients) Start(server string, id int) {
	for {
		select {
		case job := <-fcgi.workerCh:
			go func() {
				client, err := fcgiclient.Dial("tcp", server)
				job.Run(client, err)
			}()
		case <-fcgi.closing:
			return
		}
	}
}

func (fcgi *fCgiClients) Init() {
	fcgi.servers.Each(func(idx int, server string) bool {
		log.Printf("Starting FCgi: name=%s server=%s", fcgi.name, server)
		go fcgi.Start(server, idx)
		return true
	})
}
func (fcgi *fCgiClients) Kill() {
	for _, _ = range fcgi.servers {
		fcgi.closing <- true
	}
}

func (fcgi *fCgiClients) Jobs() chan *fCgiClientJob {
	return fcgi.workerCh
}

func newFcgiClient(name string, servers cfgServerList) (fcgi *fCgiClients) {
	fcgi = &fCgiClients{
		name:     name,
		servers:  servers,
		workerCh: make(chan *fCgiClientJob, len(servers)*100),
		closing:  make(chan bool, len(servers)),
	}
	fcgi.Init()
	return
}
