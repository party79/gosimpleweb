package main

import (
	"fmt"
	"io/ioutil"
	"log"

	"gopkg.in/yaml.v2"
)

type cfgProxyOpts struct {
	Server  string `yaml:"proxy_server"`
	Pattern string `yaml:"proxy_pattern"`
}

func (cfg *cfgProxyOpts) String() string {
	return fmt.Sprintf("{ server: %s, pattern: %s }", cfg.Server, cfg.Pattern)
}

type cfgProxyOptsList []*cfgProxyOpts

func (cfg cfgProxyOptsList) Each(cb func(idx int, proxyOpts *cfgProxyOpts) bool) {
	if cfg != nil {
		for idx, fCgiCfg := range cfg {
			if !cb(idx, fCgiCfg) {
				break
			}
		}
	}
}

func (cfg cfgProxyOptsList) String() string {
	ret := ""
	cfg.Each(func(idx int, proxyOpts *cfgProxyOpts) bool {
		if idx == 0 {
			ret = fmt.Sprintf("[ %s", proxyOpts)
		} else {
			ret = fmt.Sprintf("%s,  %s", ret, proxyOpts)
		}
		return true
	})
	if ret == "" {
		ret = "[]"
	} else {
		ret = fmt.Sprintf("%s ]", ret)
	}
	return ret
}

type cfgFCgiOpts struct {
	Server  string            `yaml:"fcgi_server"`
	Pattern string            `yaml:"fcgi_pattern"`
	Script  string            `yaml:"fcgi_script"`
	Index   string            `yaml:"fcgi_index"`
	Params  map[string]string `yaml:"fcgi_params"`
}

func (cfg *cfgFCgiOpts) String() string {
	return fmt.Sprintf("{ server: %s, pattern: %s, script: %s, params: %+v }", cfg.Server, cfg.Pattern, cfg.Script, cfg.Params)
}

type cfgFCgiOptsList []*cfgFCgiOpts

func (cfg cfgFCgiOptsList) Each(cb func(idx int, fCgiOpts *cfgFCgiOpts) bool) {
	if cfg != nil {
		for idx, fCgiCfg := range cfg {
			if !cb(idx, fCgiCfg) {
				break
			}
		}
	}
}

func (cfg cfgFCgiOptsList) String() string {
	ret := ""
	cfg.Each(func(idx int, fCgiOpts *cfgFCgiOpts) bool {
		if idx == 0 {
			ret = fmt.Sprintf("[ %s", fCgiOpts)
		} else {
			ret = fmt.Sprintf("%s,  %s", ret, fCgiOpts)
		}
		return true
	})
	if ret == "" {
		ret = "[]"
	} else {
		ret = fmt.Sprintf("%s ]", ret)
	}
	return ret
}

type cfgSslOpts struct {
	Key     string `yaml:"ssl_key"`
	KeyPass string `yaml:"ssl_key_pass"`
	Cert    string `yaml:"ssl_cert"`
	Chain   string `yaml:"ssl_chain"`
}

func (cfg *cfgSslOpts) String() string {
	return fmt.Sprintf("{ key: %s, keyPass: %s, cert: %s, chain: %s }", cfg.Key, cfg.KeyPass, cfg.Cert, cfg.Chain)
}

type cfgSite struct {
	Host    string           `yaml:"site_host"`
	Ip      string           `yaml:"site_ip"`
	Port    string           `yaml:"site_port"`
	Root    string           `yaml:"site_root"`
	SslOn   bool             `yaml:"site_ssl_on"`
	SslOpts *cfgSslOpts      `yaml:"site_ssl_opts"`
	FCgi    cfgFCgiOptsList  `yaml:"site_fcgi"`
	Proxy   cfgProxyOptsList `yaml:"site_proxy"`
}

func (cfg *cfgSite) Addr() string {
	return fmt.Sprintf("%s:%s", cfg.Ip, cfg.Port)
}

func (cfg *cfgSite) String() string {
	return fmt.Sprintf("{ host: %s, ip: %s, port: %s, root: %s, sslOn: %t, sslOpts: %s, fcgi: %s, proxy: %s }", cfg.Host, cfg.Ip, cfg.Port, cfg.Root, cfg.SslOn, cfg.SslOpts, cfg.FCgi, cfg.Proxy)
}

type cfgSiteList []*cfgSite

func (cfg cfgSiteList) Each(cb func(idx int, site *cfgSite) bool) {
	if cfg != nil {
		for idx, siteCfg := range cfg {
			if !cb(idx, siteCfg) {
				break
			}
		}
	}
}

func (cfg cfgSiteList) String() string {
	ret := ""
	cfg.Each(func(idx int, site *cfgSite) bool {
		if idx == 0 {
			ret = fmt.Sprintf("[ %s", site)
		} else {
			ret = fmt.Sprintf("%s,  %s", ret, site)
		}
		return true
	})
	if ret == "" {
		ret = "[]"
	} else {
		ret = fmt.Sprintf("%s ]", ret)
	}
	return ret
}

type cfgServerList []string

func (cfg cfgServerList) Each(cb func(idx int, server string) bool) {
	if cfg != nil {
		for idx, server := range cfg {
			if !cb(idx, server) {
				break
			}
		}
	}
}

func (cfg cfgServerList) String() string {
	ret := ""
	cfg.Each(func(idx int, server string) bool {
		if idx == 0 {
			ret = fmt.Sprintf("[ %s", server)
		} else {
			ret = fmt.Sprintf("%s,  %s", ret, server)
		}
		return true
	})
	if ret == "" {
		ret = "[]"
	} else {
		ret = fmt.Sprintf("%s ]", ret)
	}
	return ret
}

type cfgServerMap map[string]cfgServerList

func (cfg cfgServerMap) Each(cb func(label string, lst cfgServerList) bool) {
	if cfg != nil {
		for label, lst := range cfg {
			if !cb(label, lst) {
				break
			}
		}
	}
}
func (cfg cfgServerMap) Get(n string) (val cfgServerList, ok bool) {
	if cfg != nil {
		val, ok = cfg[n]
	}
	return
}

func (cfg cfgServerMap) String() string {
	ret := ""
	cfg.Each(func(label string, lst cfgServerList) bool {
		if ret == "" {
			ret = fmt.Sprintf("[ %s: %s", label, lst)
		} else {
			ret = fmt.Sprintf("%s, %s: %s", ret, label, lst)
		}
		return true
	})
	if ret == "" {
		ret = "{}"
	} else {
		ret = fmt.Sprintf("%s }", ret)
	}
	return ret
}

type config struct {
	Sites        cfgSiteList  `yaml:"sites"`
	FCgiServers  cfgServerMap `yaml:"fcgi"`
	ProxyServers cfgServerMap `yaml:"proxy"`
	Live         bool         `yaml:"live"`
}

func (cfg *config) String() string {
	return fmt.Sprintf("{ sites: %s, fcgi_servers: %+v, proxy_servers: %+v, live: %t }", cfg.Sites, cfg.FCgiServers, cfg.ProxyServers, cfg.Live)
}

func loadConfig() (cfg *config) {
	file, e := ioutil.ReadFile("config.yml")
	if e != nil {
		log.Fatalf("Config file error: %v\n", e)
	}
	var err error

	cfg = new(config)
	err = yaml.Unmarshal(file, cfg)
	if err != nil {
		log.Fatalf("Config file error: %v\n", err)
	}

	return
}
