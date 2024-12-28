package main

import (
	"fmt"
	"net/http"
	"reverse-proxy/routes" // Replace with your module name
)

func main() {
	// Initialize routes
	router := routes.InitializeRoutes()

	// Start the server
	port := ":8080"
	fmt.Printf("Reverse Proxy Server is running on http://localhost%s\n", port)
	if err := http.ListenAndServe(port, router); err != nil {
		panic(err)
	}
}