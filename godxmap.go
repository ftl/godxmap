// The package godxmap provides a simple way to integrate F5UII's HamDXMap into Go programs.
// This allows to show callsigns, dx spots or gab chat messages (a Win-Test peculiarity) on the map.
//
// For more information about the used protocol, please refer to the [wtSock API reference]
//
// [wtSock API reference] : https://dxmap.f5uii.net/help/index.html.
package godxmap

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"golang.org/x/net/websocket"
)

const (
	writeTimeout = 100 * time.Millisecond
)

type frame map[string]any

// Server opens a websocket and allows to send wtSock frames to all connected websocket clients.
type Server struct {
	addr     string
	server   *http.Server
	inbound  chan frame
	register chan dxmapConnection
	closed   chan struct{}
}

// NewServer creates a new server instance for the given listening address. To actually start the server instance, use the Serve method.
func NewServer(addr string) *Server {
	result := &Server{
		addr:     addr,
		inbound:  make(chan frame, 1),
		register: make(chan dxmapConnection, 1),
		closed:   make(chan struct{}),
	}

	go result.run()

	return result
}

// Close the active connections, all active net.Listeners, and stop the server.
//
// Close returns any error returned from closing the [Server]'s underlying Listener(s).
func (s *Server) Close() error {
	close(s.inbound)
	<-s.closed
	return s.server.Close()
}

// Serve starts this server on its dedicated listening address.
// It accepts incoming websocket connections and will distribute wtSock frames to all connected clients.
//
// Serve always returns a non-nil error.
// After [Server.Shutdown] or [Server.Close], the returned error is [ErrServerClosed].
func (s *Server) Serve() error {
	mux := http.NewServeMux()
	mux.Handle("/", websocket.Handler(func(conn *websocket.Conn) {
		s.serveConnection(conn)
	}))

	listener, err := net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("cannot open listener: %v", err)
	}
	s.server = &http.Server{
		Handler: mux,
	}

	return s.server.Serve(listener)
}

func (s *Server) serveConnection(conn *websocket.Conn) {
	c := newDXMapConnection(conn)
	s.register <- c
	c.Serve()
}

func (s *Server) run() {
	defer close(s.closed)

	outbound := make([]dxmapConnection, 0)
	for {
		select {
		case frame, active := <-s.inbound:
			for _, c := range outbound {
				if active {
					err := c.Send(frame)
					if err != nil {
						c.Close()
					}
				} else {
					c.Close()
				}
			}
			if !active {
				return
			}
		case c := <-s.register:
			outbound = append(outbound, c)
		}
	}
}

func (s *Server) send(f frame) {
	s.inbound <- f
}

// ShowLoggedCall adds information about a logged callsign to the map.
func (s *Server) ShowLoggedCall(call string, frequencyKHz float64) {
	s.send(s.loggedCallFrame(call, frequencyKHz))
}

// ShowPartialCall shows the position of a (partially) entered callsign on the map.
func (s *Server) ShowPartialCall(call string) {
	s.send(s.partialCallFrame(call))
}

// ShowDXSpot adds information about a DX spot to the map.
func (s *Server) ShowDXSpot(spot string, spotter string, frequencyKHz float64, comments string) {
	s.send(s.dxSpotFrame(spot, spotter, frequencyKHz, comments))
}

// ShowGab displays a gab chat message next to the map.
func (s *Server) ShowGab(from string, to string, message string) {
	s.send(s.gabFrame(from, to, message))
}

func (s *Server) loggedCallFrame(call string, frequencyKHz float64) frame {
	result := s.newFrame("LoggedCall")
	result["Call"] = call
	result["Frequency"] = frequencyKHz
	return result
}

func (s *Server) partialCallFrame(call string) frame {
	result := s.newFrame("PartialCall")
	result["Call"] = call
	return result
}

func (s *Server) dxSpotFrame(spot string, spotter string, frequencyKHz float64, comments string) frame {
	result := s.newFrame("DXSpot")
	result["Spot"] = spot
	result["Spotter"] = spotter
	result["Frequency"] = frequencyKHz
	result["Comments"] = comments
	return result
}

func (s *Server) gabFrame(from string, to string, message string) frame {
	result := s.newFrame("Gab")
	result["From"] = from
	result["To"] = to
	result["Message"] = message
	return result
}

func (s *Server) newFrame(frameType string) frame {
	return frame{
		"Frame":      frameType,
		"DateTime":   time.Now().UnixMilli(),
		"SourceAddr": s.addr,
	}
}

type dxmapConnection struct {
	conn   *websocket.Conn
	closed chan struct{}
	frames chan frame
}

func newDXMapConnection(conn *websocket.Conn) dxmapConnection {
	return dxmapConnection{
		conn:   conn,
		closed: make(chan struct{}),
		frames: make(chan frame, 1),
	}
}

func (c dxmapConnection) Serve() {
	<-c.closed
}

func (c dxmapConnection) Close() error {
	select {
	case <-c.closed:
		return nil
	default:
		// go on
	}

	err := c.conn.Close()
	close(c.closed)
	return err
}

func (c dxmapConnection) Send(f frame) error {
	select {
	case <-c.closed:
		return nil
	default:
		// go on
	}

	err := c.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
	if err != nil {
		log.Printf("cannot set write deadline: %v", err)
		return err
	}

	err = websocket.JSON.Send(c.conn, f)
	if err != nil {
		log.Printf("cannot send frame: %v", err)
		return err
	}

	return nil
}
