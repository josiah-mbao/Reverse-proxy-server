package main

import (
	"fmt"
	"net/http"
)

func server() {
	// Serves dynamic content
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "The server that served you this page was built by Josiah, using Go.")
	})

	// Serve static files from folder called "static"
	fs := http.FileServer(http.Dir("static"))
	http.Handle("/static", http.StripPrefix("/static", fs))

	// Start the server
	fmt.Println("Server running on http://localhost:8080")
	http.ListenAndServe(":8080", nil)
}

func main() {
	server()
}
