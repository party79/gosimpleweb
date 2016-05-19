# gosimpleweb
Simple go based web server with fcgi and reverse proxy support

### Install
```
go get github.com/party79/gosimpleweb
```
### Examples
config.yml:
```
sites:
    - site_host: "www.default.com"
      site_ip: "www.default.com"
      site_port: 80
      site_root: "/opt/web/www/default/frontend"
      site_fcgi:
          - fcgi_server: "php"
            fcgi_pattern: "^/.+\\.php"
            fcgi_script: "/opt/web/www/default/frontend%s"
            fcgi_index: "index.php"
          - fcgi_server: "php"
            fcgi_pattern: "^/api"
            fcgi_script: "/opt/web/www/default/api%s"
            fcgi_index: "index.php"
    - site_host: "www.default.com"
      site_ip: "www.default.com"
      site_port: 443
      site_root: "/opt/web/www/default/frontend"
      site_ssl_on: true
      site_ssl_opts:
          ssl_key: "/opt/web/certs/www.default.com.key"
          ssl_cert: "/opt/web/certs/www.default.com.crt"
      site_fcgi:
          - fcgi_server: "php"
            fcgi_pattern: "^/.+\\.php"
            fcgi_script: "/opt/web/www/default/frontend%s"
            fcgi_index: "index.php"
          - fcgi_server: "php"
            fcgi_pattern: "^/api"
            fcgi_script: "/opt/web/www/default/api%s"
            fcgi_index: "index.php"
fcgi:
    php:
        - "127.0.0.1:9000"
live: true
```
