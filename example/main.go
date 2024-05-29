package main

import (
	"log"
	"net/http"
	"time"

	"github.com/ftl/godxmap"
)

func main() {
	log.Printf("goDXMap")

	server := godxmap.NewServer(":12345")

	go serveCallsigns(server)

	err := server.Serve()
	if err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func serveCallsigns(server *godxmap.Server) {
	time.Sleep(10 * time.Second)
	server.ShowGab("goDXMap", "HamDXMap", "Starting the demo...")

	callsigns := []string{"F5UII", "W1AW", "PY1PY", "DL3NEY", "ZL2CTM"}
	for _, callsign := range callsigns {
		time.Sleep(10 * time.Second)
		server.ShowPartialCall(callsign)
	}

	time.Sleep(10 * time.Second)
	server.ShowGab("goDXMap", "HamDXMap", "That was the demo. 73!")

	err := server.Close()
	if err != nil {
		log.Printf("error closing the server: %v", err)
	}
}
