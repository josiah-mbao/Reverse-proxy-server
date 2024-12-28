package handlers

import (
	"net/http"
	"net/http/httputil"
	"net/url"
)

// ProxyHandler sets up a reverse proxy to the specified target.
func ProxyHandler(target string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		remote, err := url.Parse(target)
		if err != nil {
			http.Error(w, "Invalid target URL", http.StatusInternalServerError)
			return
		}

		proxy := httputil.NewSingleHostReverseProxy(remote)
		r.Host = remote.Host
		proxy.ServeHTTP(w, r)
	}
}