package main

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
)

func reverseProxy(target string) http.Handler {
	targetURL, err := url.Parse(target)
	if err != nil {
		fmt.Println("Error parsing URL: ", err)
		return nil
	}
	return httputil.NewSingleHostReverseProxy(targetURL)
}

func main() {
	backend := "http://127.0.0.1:5000" // Plate planner backend
	http.Handle("/", reverseProxy(backend))
	fmt.Println("Aight, starting server on :8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		fmt.Println("Damn, the server failed to start:", err)
	}
}
