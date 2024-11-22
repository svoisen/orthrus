package web

import (
	"fmt"
	"net/http"
)

type WebServerConfig struct {
	ContentDir string
	Port       int
}

type WebServer struct {
	Config WebServerConfig
}

func NewWebServer(cfg WebServerConfig) *WebServer {
	server := &WebServer{
		Config: cfg,
	}

	return server
}

func (s *WebServer) Start() error {
	fmt.Println("web server listening on port:", s.Config.Port)
	address := fmt.Sprintf("localhost: %v", s.Config.Port)
	http.Handle("/", http.FileServer(http.Dir(s.Config.ContentDir)))
	if err := http.ListenAndServe(address, nil); err != nil {
		return err
	}

	return nil
}
