package main

import (
	"flag"
	"github.com/burakkarasel/Chat-App/internal/handlers"
	"log"
	"net/http"
)

func main() {
	port := flag.String("port", "Port Value for the app", "Defines the port of the app")
	flag.Parse()

	mux := routes()

	log.Println("Starting web server")
	log.Println("Starting web socket channel listener")
	go handlers.ListenToWsChannel()

	err := http.ListenAndServe(*port, mux)

	if err != nil {
		log.Fatal(err)
	}
}
