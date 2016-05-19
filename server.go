package main

type server struct {
	cfg             *config
	running         bool
	fCgiClientsMap  map[string]*fCgiClients
	proxyClientsMap map[string]*proxyClients
	listeners       map[string]listener
	Stopped         chan bool
}

func (srv *server) GetCfg() *config {
	return srv.cfg
}
func (srv *server) GetFcgi(name string) (client *fCgiClients, ok bool) {
	client, ok = srv.fCgiClientsMap[name]
	return
}
func (srv *server) GetProxy(name string) (client *proxyClients, ok bool) {
	client, ok = srv.proxyClientsMap[name]
	return
}

func (srv *server) Start() {
	if srv.running {
		return
	}
	srv.cfg.FCgiServers.Each(func(label string, lst cfgServerList) bool {
		srv.fCgiClientsMap[label] = newFcgiClient(label, lst)
		return true
	})
	srv.cfg.ProxyServers.Each(func(label string, lst cfgServerList) bool {
		srv.proxyClientsMap[label] = newProxyClient(label, lst)
		return true
	})
	srv.cfg.Sites.Each(func(idx int, site *cfgSite) bool {
		laddr := site.Addr()
		if lstnr, ok := srv.listeners[laddr]; ok {
			lstnr.AddSite(site)
		} else {
			if site.SslOn {
				lstnr = newHttpsListener(srv, laddr, site.SslOpts)
			} else {
				lstnr = newHttpListener(srv, laddr)
			}
			lstnr.AddSite(site)
			srv.listeners[laddr] = lstnr
		}
		return true
	})
	for _, lstnr := range srv.listeners {
		go lstnr.Open()
	}
	srv.running = true
}
func (srv *server) Stop() {
	if !srv.running {
		return
	}
	srv.running = false
	for _, lstnr := range srv.listeners {
		go lstnr.Close()
		<-lstnr.ClosedCh()
	}
	srv.listeners = make(map[string]listener)
	for _, fcgi := range srv.fCgiClientsMap {
		fcgi.Kill()
	}
	srv.fCgiClientsMap = make(map[string]*fCgiClients)
	srv.Stopped <- true
}

func newServer(cfg *config) (srv *server) {
	srv = &server{
		cfg:            cfg,
		running:        false,
		fCgiClientsMap: make(map[string]*fCgiClients),
		listeners:      make(map[string]listener),
		Stopped:        make(chan bool, 1),
	}
	go srv.Start()
	return
}
