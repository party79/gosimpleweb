package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"time"
)

type listener interface {
	AddSite(site *cfgSite)
	Open()
	Close()
	IsOpen() bool
	ClosedCh() chan bool
}

type tcpKeepAliveListener struct {
	*net.TCPListener
}

func (ln tcpKeepAliveListener) Accept() (c net.Conn, err error) {
	tc, err := ln.AcceptTCP()
	if err != nil {
		return
	}
	tc.SetKeepAlive(true)
	tc.SetKeepAlivePeriod(3 * time.Minute)
	return tc, nil
}

func addSite(srv *server, srvMux *serveMux, laddr string, site *cfgSite, mapDefault bool) {
	if mapDefault {
		log.Printf("Adding Site: host=%s laddr=%s, root=%s", "default", laddr, site.Root)
		if site.Root != "" {
			srvMux.Handle("/", http.FileServer(http.Dir(site.Root)))
		}
		site.FCgi.Each(func(idx int, fCgiOpts *cfgFCgiOpts) bool {
			if fCgiClients, ok := srv.GetFcgi(fCgiOpts.Server); ok {
				log.Printf("Adding Site FCgi: host=%s laddr=%s, fcgi_server=%s, path_pattern=%s", "default", laddr, fCgiOpts.Server, fCgiOpts.Pattern)
				srvMux.HandleMatch("", fCgiOpts.Pattern, &fCgiHandler{clients: fCgiClients, fCfg: fCgiOpts, pCfg: srv.GetCfg(), laddr: laddr})
			}
			return true
		})
		site.Proxy.Each(func(idx int, proxyOpts *cfgProxyOpts) bool {
			if proxyClients, ok := srv.GetProxy(proxyOpts.Server); ok {
				log.Printf("Adding Site FCgi: host=%s laddr=%s, fcgi_server=%s, path_pattern=%s", "default", laddr, proxyOpts.Server, proxyOpts.Pattern)
				srvMux.HandleMatch("", proxyOpts.Pattern, proxyClients)
			}
			return true
		})
	}
	log.Printf("Adding Site: host=%s laddr=%s, root=%s", site.Host, laddr, site.Root)
	if site.Root != "" {
		srvMux.Handle(fmt.Sprintf("%s/", site.Host), http.FileServer(http.Dir(site.Root)))
	}
	site.FCgi.Each(func(idx int, fCgiOpts *cfgFCgiOpts) bool {
		if fCgiClients, ok := srv.GetFcgi(fCgiOpts.Server); ok {
			log.Printf("Adding Site FCgi: host=%s laddr=%s, fcgi_server=%s, path_pattern=%s", site.Host, laddr, fCgiOpts.Server, fCgiOpts.Pattern)
			srvMux.HandleMatch(site.Host, fCgiOpts.Pattern, &fCgiHandler{clients: fCgiClients, fCfg: fCgiOpts, pCfg: srv.GetCfg(), laddr: laddr})
		}
		return true
	})
	site.Proxy.Each(func(idx int, proxyOpts *cfgProxyOpts) bool {
		if proxyClients, ok := srv.GetProxy(proxyOpts.Server); ok {
			log.Printf("Adding Site FCgi: host=%s laddr=%s, fcgi_server=%s, path_pattern=%s", site.Host, laddr, proxyOpts.Server, proxyOpts.Pattern)
			srvMux.HandleMatch(site.Host, proxyOpts.Pattern, proxyClients)
		}
		return true
	})
}
