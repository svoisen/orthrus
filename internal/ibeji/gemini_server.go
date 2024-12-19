package ibeji

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"git.sr.ht/~adnano/go-gemini"
	"git.sr.ht/~adnano/go-gemini/certificate"
)

type GeminiServer struct {
	Config       GeminiConfig
	Certificates certificate.Store
}

// NewGeminiServer creates a new Gemini server
func NewGeminiServer(cfg GeminiConfig) *GeminiServer {
	server := &GeminiServer{
		Config: cfg,
	}

	return server
}

// Start starts the Gemini server
func (s *GeminiServer) Start() error {
	var server gemini.Server
	server.ReadTimeout = 1 * time.Minute
	server.WriteTimeout = 2 * time.Minute

	err := s.Certificates.Load(s.Config.CertStore)
	if err != nil {
		fmt.Println("unable to load certificate:", err)
		return err
	}

	s.Certificates.Register("*." + s.Config.Hostname)
	s.Certificates.Register(s.Config.Hostname)
	server.GetCertificate = s.Certificates.Get

	var mux gemini.Mux
	mux.HandleFunc("/", s.getGeminiPage)
	server.Handler = gemini.LoggingMiddleware(&mux)

	log.Println("gemini server listening on port:", s.Config.Port)
	err = server.ListenAndServe(context.Background())
	if err != nil {
		fmt.Println("error starting gemini server", err)
		return err
	}

	return nil
}

// getGeminiPage acts as the handler function for all requests
func (s *GeminiServer) getGeminiPage(_ context.Context, w gemini.ResponseWriter, r *gemini.Request) {
	gemini.ServeFile(w, os.DirFS(s.Config.OutputDir), r.URL.Path)
}
