package godxmap

import (
	"context"
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

type Server struct {
	addr     string
	server   *http.Server
	inbound  chan frame
	register chan chan<- frame
}

func NewServer(addr string) *Server {
	result := &Server{
		addr:     addr,
		inbound:  make(chan frame),
		register: make(chan chan<- frame),
	}

	go result.run()

	return result
}

func (s *Server) Serve() error {
	mux := http.NewServeMux()
	mux.Handle("/", websocket.Handler(func(conn *websocket.Conn) {
		s.serveFrames(conn)
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

func (s *Server) Close() error {
	close(s.inbound)
	return s.server.Close()
}

func (s *Server) Shutdown(ctx context.Context) error {
	close(s.inbound)
	return s.server.Shutdown(ctx)
}

func (s *Server) serveFrames(conn *websocket.Conn) {
	frames := make(chan frame, 1)
	s.register <- frames

	for frame := range frames {
		err := conn.SetWriteDeadline(time.Now().Add(writeTimeout))
		if err != nil {
			log.Printf("cannot set write deadline: %v", err)
			close(frames)
			return
		}

		err = websocket.JSON.Send(conn, frame)
		if err != nil {
			log.Printf("cannot send frame: %v", err)
			close(frames)
			return
		}
	}
}

func (s *Server) run() {
	outbound := make([]chan<- frame, 0)
	fanOut := func(c chan<- frame, f frame) bool {
		select {
		case c <- f:
			return true
		default:
			return false
		}
	}

	for {
		select {
		case frame, active := <-s.inbound:
			for _, c := range outbound {
				if active {
					fanOut(c, frame)
				} else {
					close(c)
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

func (s *Server) ShowLoggedCall(call string, frequencyKHz float64) {
	s.send(s.loggedCallFrame(call, frequencyKHz))
}

func (s *Server) ShowPartialCall(call string) {
	s.send(s.partialCallFrame(call))
}

func (s *Server) ShowDXSpot(spot string, spotter string, frequencyKHz float64, comments string) {
	s.send(s.dxSpotFrame(spot, spotter, frequencyKHz, comments))
}

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
	result := s.newFrame("PartialCall")
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
