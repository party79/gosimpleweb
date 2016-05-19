package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
)

func cloneTLSConfig(cfg *tls.Config) *tls.Config {
	if cfg == nil {
		return &tls.Config{}
	}
	return &tls.Config{
		Rand:                     cfg.Rand,
		Time:                     cfg.Time,
		Certificates:             cfg.Certificates,
		NameToCertificate:        cfg.NameToCertificate,
		GetCertificate:           cfg.GetCertificate,
		RootCAs:                  cfg.RootCAs,
		NextProtos:               cfg.NextProtos,
		ServerName:               cfg.ServerName,
		ClientAuth:               cfg.ClientAuth,
		ClientCAs:                cfg.ClientCAs,
		InsecureSkipVerify:       cfg.InsecureSkipVerify,
		CipherSuites:             cfg.CipherSuites,
		PreferServerCipherSuites: cfg.PreferServerCipherSuites,
		SessionTicketsDisabled:   cfg.SessionTicketsDisabled,
		SessionTicketKey:         cfg.SessionTicketKey,
		ClientSessionCache:       cfg.ClientSessionCache,
		MinVersion:               cfg.MinVersion,
		MaxVersion:               cfg.MaxVersion,
		CurvePreferences:         cfg.CurvePreferences,
	}
}
func strSliceContains(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
}

type httpsListener struct {
	srv      *server
	running  bool
	laddr    string
	sslOpts  *cfgSslOpts
	closing  chan bool
	Closed   chan bool
	listener net.Listener
	srvMux   *serveMux
}

func (lstnr *httpsListener) AddSite(site *cfgSite) {
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

func (lstnr *httpsListener) Open() {
	if lstnr.running {
		return
	}
	lstnr.running = true
	errch := make(chan error, 1)
	go func() {
		var err error
		certPEMBlock := make([]byte, 0)
		keyPEMBlock := make([]byte, 0)
		if f := lstnr.sslOpts.Chain; f != "" {
			if v, err := ioutil.ReadFile(f); err == nil {
				certPEMBlock = append(certPEMBlock, v...)
			} else {
				log.Fatal(err)
			}
		}
		if v, err := ioutil.ReadFile(lstnr.sslOpts.Cert); err == nil {
			certPEMBlock = append(certPEMBlock, v...)
		} else {
			log.Fatal(err)
		}
		if v, err := ioutil.ReadFile(lstnr.sslOpts.Key); err == nil {
			keyPEMBlock = append(keyPEMBlock, v...)
		} else {
			log.Fatal(err)
		}
		passphrase := lstnr.sslOpts.KeyPass
		priv, _ := pem.Decode(keyPEMBlock)
		if priv == nil {
			log.Fatal("Key file error: no pem data")
		}
		if x509.IsEncryptedPEMBlock(priv) {
			if passphrase == "" {
				log.Fatal("Key file error: no pass phrase given")
			}
			b, err := x509.DecryptPEMBlock(priv, []byte(passphrase))
			if err != nil {
				log.Fatal(fmt.Sprintf("Key file error: %v", err))
			}
			keyPEMBlock = pem.EncodeToMemory(&pem.Block{
				Type:  "RSA PRIVATE KEY",
				Bytes: b,
			})
		} else if passphrase != "" {
			log.Fatal("Key file error: invalid pass phrase given")
		}
		myCerts := make([]tls.Certificate, 1)
		if myCerts[0], err = tls.X509KeyPair(certPEMBlock, keyPEMBlock); err != nil {
			log.Fatal(err)
		}
		myTLSConfig := &tls.Config{
			Certificates: myCerts,
			MinVersion:   tls.VersionTLS12,
			CipherSuites: []uint16{
				tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
				tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
				tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
				tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
				tls.TLS_RSA_WITH_AES_128_CBC_SHA,
				tls.TLS_RSA_WITH_AES_256_CBC_SHA,
			},
		}
		myTLSConfig.PreferServerCipherSuites = true
		myTLSWebServer := &http.Server{Addr: lstnr.laddr, TLSConfig: myTLSConfig, Handler: lstnr.srvMux}

		cfg := cloneTLSConfig(myTLSWebServer.TLSConfig)
		if !strSliceContains(cfg.NextProtos, "http/1.1") {
			cfg.NextProtos = append(cfg.NextProtos, "http/1.1")
		}

		lstnr.listener, err = net.Listen("tcp", lstnr.laddr)
		if err != nil {
			errch <- err
		} else {
			tlsListener := tls.NewListener(tcpKeepAliveListener{lstnr.listener.(*net.TCPListener)}, cfg)
			errch <- myTLSWebServer.Serve(tlsListener)
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
func (lstnr *httpsListener) Close() {
	if !lstnr.running {
		return
	}
	lstnr.closing <- true
	lstnr.running = false
	lstnr.listener.Close()
	lstnr.Closed <- true
}
func (lstnr *httpsListener) IsOpen() bool {
	return lstnr.running
}
func (lstnr *httpsListener) ClosedCh() chan bool {
	return lstnr.Closed
}

func newHttpsListener(srv *server, laddr string, sslOpts *cfgSslOpts) (lstnr *httpsListener) {
	lstnr = &httpsListener{
		srv:     srv,
		running: false,
		laddr:   laddr,
		sslOpts: sslOpts,
		closing: make(chan bool, 1),
		Closed:  make(chan bool, 1),
	}
	return
}
