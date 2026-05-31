package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/shhac/agent-postmark/internal/mockpostmark"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:12122", "listen address")
	routes := flag.Bool("routes", false, "print routes and exit")
	flag.Parse()

	if *routes {
		for _, route := range mockpostmark.Routes() {
			fmt.Println(route)
		}
		return
	}

	log.Printf("mockpostmark listening on http://%s", *addr)
	log.Fatal(http.ListenAndServe(*addr, mockpostmark.NewServer()))
}
