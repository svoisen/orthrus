package orthrus

import (
	"fmt"
	"net/http"
)

type WebServer struct {
	Config WebConfig
}

// NewWebServer creates a new web server
func NewWebServer(cfg WebConfig) *WebServer {
	server := &WebServer{
		Config: cfg,
	}

	return server
}

// Start starts the web server
func (s *WebServer) Start() error {
	fmt.Println("web server listening on port:", s.Config.Port)

	address := fmt.Sprintf("localhost: %v", s.Config.Port)
	http.Handle("/", http.FileServer(http.Dir(s.Config.OutputDir)))
	if err := http.ListenAndServe(address, nil); err != nil {
		return err
	}

	return nil
}
