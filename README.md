# goDXMap
This is a simple Go library to integrate with [F5UUI's HamDXMap](https://dxmap.f5uii.net/). It opens a websocket and allows to display different things on the map, using the [wtSock protocol](https://dxmap.f5uii.net/help/index.html).

## Example

```go
// see also ./example/main.go
func main() {
	server := godxmap.NewServer(":12345")

	go serveCallsigns(server)

	err := server.Serve()
	if err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func serveCallsigns(server *godxmap.Server) {
	callsigns := []string{"F5UII", "W1AW", "PY1PY", "DL3NEY", "ZL2CTM"}
	for _, callsign := range callsigns {
		time.Sleep(10 * time.Second)
		server.ShowPartialCall(callsign)
	}
	err := server.Close()
	if err != nil {
		log.Printf("error closing the server: %v", err)
	}
}
```

## License
This library is published under the [MIT License](https://www.tldrlegal.com/l/mit).

Copyright [Florian Thienel](http://thecodingflow.com/)
