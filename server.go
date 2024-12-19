package main

import (
	"fmt"
	"net/http"
)

func main() {
	// Our server does 3 things: It handles dynamic requests, serves static content and accepts connections
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "The server that served you this page was built by Josiah, using Go.")
	})

	fs := http.FileServer(http.Dir("/static"))
	http.Handle("/static", http.StripPrefix("/static", fs))

	http.ListenAndServe(":8080", nil)
}
