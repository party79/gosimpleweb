package main

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/tomasen/fcgi_client"
)

var cookieNameSanitizer = strings.NewReplacer("\n", "-", "\r", "-")

func sanitizeCookieName(n string) string {
	return cookieNameSanitizer.Replace(n)
}

func sanitizeCookieValue(v string) string {
	v = sanitizeOrWarn("Cookie.Value", validCookieValueByte, v)
	if len(v) == 0 {
		return v
	}
	if v[0] == ' ' || v[0] == ',' || v[len(v)-1] == ' ' || v[len(v)-1] == ',' {
		return `"` + v + `"`
	}
	return v
}
func validCookieValueByte(b byte) bool {
	return 0x20 <= b && b < 0x7f && b != '"' && b != ';' && b != '\\'
}

func sanitizeOrWarn(fieldName string, valid func(byte) bool, v string) string {
	ok := true
	for i := 0; i < len(v); i++ {
		if valid(v[i]) {
			continue
		}
		log.Printf("net/http: invalid byte %q in %s; dropping invalid bytes", v[i], fieldName)
		ok = false
		break
	}
	if ok {
		return v
	}
	buf := make([]byte, 0, len(v))
	for i := 0; i < len(v); i++ {
		if b := v[i]; valid(b) {
			buf = append(buf, b)
		}
	}
	return string(buf)
}

type fCgiHandler struct {
	clients *fCgiClients
	fCfg    *cfgFCgiOpts
	pCfg    *config
	laddr   string
}

func (hndlr *fCgiHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	errch := make(chan error, 1)
	hndlr.clients.Jobs() <- &fCgiClientJob{
		process: func(fcgi *fcgiclient.FCGIClient, err error) {
			defer func() {
				if r := recover(); r != nil {
					if r != nil {
						switch x := r.(type) {
						case runtime.Error:
							log.Panicln(r)
						case string:
							errch <- errors.New(x)
						case error:
							errch <- x
						default:
							errch <- errors.New("Unknown panic")
						}
					}
				}
			}()

			if fcgi == nil {
				errch <- errors.New("No fast cgi available")
				return
			}

			remoteAddr := strings.SplitN(req.RemoteAddr, ":", 2)
			serverAddr := strings.SplitN(hndlr.laddr, ":", 2)
			fileName := fmt.Sprintf(hndlr.fCfg.Script, req.URL.Path)
			if hndlr.fCfg.Index != "" {
				n := len(fileName)
				if fileName[n-1] == '/' {
					fileName = fileName + hndlr.fCfg.Index
				}
			}

			params := make(map[string]string)
			params["SCRIPT_FILENAME"] = fileName
			params["SERVER_SOFTWARE"] = "goweb/1.0"
			params["REQUEST_URI"] = req.RequestURI
			params["REQUEST_METHOD"] = req.Method
			params["SERVER_PROTOCOL"] = req.Proto
			params["REMOTE_ADDR"] = remoteAddr[0]
			params["REMOTE_PORT"] = remoteAddr[1]
			params["SERVER_ADDR"] = serverAddr[0]
			params["SERVER_PORT"] = serverAddr[1]
			for k, v := range req.Header {
				if len(v) > 0 {
					pk := fmt.Sprintf("HTTP_%s", strings.ToUpper(strings.Replace(k, "-", "_", 0)))
					if _, ok := params[pk]; !ok {
						params[pk] = strings.Join(v, "; ")
					}
				}
			}
			if _, ok := params["HTTP_HOST"]; !ok {
				params["HTTP_HOST"] = serverAddr[0]
			}
			if _, ok := params["HTTP_CONNECTION"]; !ok {
				params["HTTP_CONNECTION"] = "keep-alive"
			}
			if _, ok := params["HTTP_COOKIE"]; !ok {
				cookies := ""
				for i, cookie := range req.Cookies() {
					cookieStr := fmt.Sprintf("%s=%s", sanitizeCookieName(cookie.Name), sanitizeCookieValue(cookie.Value))
					if i == 0 {
						cookies = cookieStr
					} else {
						cookies = fmt.Sprintf("%s; %s", cookies, cookieStr)
					}
				}
				if cookies != "" {
					params["HTTP_COOKIE"] = cookies
				}
			}
			params["REQUEST_TIME"] = strconv.Itoa(int(time.Now().Unix()))
			for k, v := range hndlr.fCfg.Params {
				params[k] = v
			}
			var body io.Reader
			if req.Method == "HEAD" || req.Method == "GET" || req.Method == "OPTIONS" || req.Method == "DELTET" {
				params["QUERY_STRING"] = req.URL.RawQuery
			}
			if req.Method == "OPTIONS" || req.Method == "PUT" || req.Method == "DELTET" || req.Method == "PATCH" || req.Method == "POST" {
				if v := req.Header.Get("Content-Type"); v != "" {
					params["CONTENT_TYPE"] = v
				} else {
					params["CONTENT_TYPE"] = "application/x-www-form-urlencoded"
				}
				if v := req.Header.Get("Content-Length"); v != "" {
					if v2, err := strconv.Atoi(v); err == nil {
						params["CONTENT_LENGTH"] = strconv.Itoa(v2)
					}
				}
				body = req.Body
			}
			if resp, err := fcgi.Request(params, body); err == nil {
				if content, err := ioutil.ReadAll(resp.Body); err == nil {
					for k, v := range resp.Header {
						for i, v2 := range v {
							if i == 0 {
								if k == "Status" && resp.StatusCode == 0 {
									status := strings.SplitN(v2, " ", 2)
									if v3, err := strconv.Atoi(status[0]); err == nil {
										resp.StatusCode = v3
									}
								}
								rw.Header().Set(k, v2)
							} else {
								rw.Header().Add(k, v2)
							}
						}
					}
					if resp.StatusCode != 0 {
						rw.WriteHeader(resp.StatusCode)
					}
					if _, err := rw.Write(content); err == nil {
						errch <- nil
					} else {
						errch <- err
					}
				} else {
					errch <- err
				}
			} else {
				errch <- err
			}
		},
	}
	if err := <-errch; err != nil {
		if err != io.EOF && err.Error() != "No fast cgi available" {
			log.Println("fcgi-err:", err)
		}
		rw.Header().Set("", "text/plain")
		rw.WriteHeader(http.StatusInternalServerError)
		body := "500: Internal Server Error"
		if !hndlr.pCfg.Live {
			body += "\n" + err.Error()
		}
		rw.Write([]byte(body))
	}
}
