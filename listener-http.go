package main

import (
	"log"
	"net"
	"net/http"
)

type httpListener struct {
	srv      *server
	running  bool
	laddr    string
	closing  chan bool
	Closed   chan bool
	listener net.Listener
	srvMux   *serveMux
}

func (lstnr *httpListener) AddSite(site *cfgSite) {
	if lstnr.running {
		return
	}
	if lstnr.srvMux == nil {
		lstnr.srvMux = newServeMux()
		addSite(lstnr.srv, lstnr.srvMux, lstnr.laddr, site, true)
	} else {
		addSite(lstnr.srv, lstnr.srvMux, lstnr.laddr, site, false)
	}
}

func (lstnr *httpListener) Open() {
	if lstnr.running {
		return
	}
	lstnr.running = true
	errch := make(chan error, 1)
	go func() {
		var err error
		lstnr.listener, err = net.Listen("tcp", lstnr.laddr)
		if err != nil {
			errch <- err
		} else {
			errch <- http.Serve(tcpKeepAliveListener{lstnr.listener.(*net.TCPListener)}, lstnr.srvMux)
		}
	}()
	select {
	case err := <-errch:
		if err != nil && lstnr.running {
			log.Fatal(err)
		}
	case <-lstnr.closing:
	}
}
func (lstnr *httpListener) Close() {
	if !lstnr.running {
		return
	}
	lstnr.closing <- true
	lstnr.running = false
	lstnr.listener.Close()
	lstnr.Closed <- true
}
func (lstnr *httpListener) IsOpen() bool {
	return lstnr.running
}
func (lstnr *httpListener) ClosedCh() chan bool {
	return lstnr.Closed
}

func newHttpListener(srv *server, laddr string) (lstnr *httpListener) {
	lstnr = &httpListener{
		srv:     srv,
		running: false,
		laddr:   laddr,
		closing: make(chan bool, 1),
		Closed:  make(chan bool, 1),
	}
	return
}
