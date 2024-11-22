package gemini

import (
	"context"
	"fmt"
	"os"
	"time"

	gemini "git.sr.ht/~adnano/go-gemini"
	"git.sr.ht/~adnano/go-gemini/certificate"
)

type GeminiServerConfig struct {
	ContentDir string
	HostName   string
	CertStore  string
	Port       int
}

type GeminiServer struct {
	Config       GeminiServerConfig
	Certificates certificate.Store
}

func NewGeminiServer(cfg GeminiServerConfig) *GeminiServer {
	server := &GeminiServer{
		Config: cfg,
	}

	return server
}

func (s *GeminiServer) Start() error {
	var server gemini.Server
	server.ReadTimeout = 1 * time.Minute
	server.WriteTimeout = 2 * time.Minute

	err := s.Certificates.Load(s.Config.CertStore)
	if err != nil {
		fmt.Println("unable to load certificate:", err)
		return err
	}

	s.Certificates.Register("*." + s.Config.HostName)
	s.Certificates.Register(s.Config.HostName)
	server.GetCertificate = s.Certificates.Get

	var mux gemini.Mux
	mux.HandleFunc("/", s.getGeminiPage)
	server.Handler = gemini.LoggingMiddleware(&mux)

	fmt.Println("gemini server listening on port:", s.Config.Port)
	err = server.ListenAndServe(context.Background())
	if err != nil {
		fmt.Println("error starting gemini server", err)
		return err
	}

	return nil
}

func (s *GeminiServer) getGeminiPage(_ context.Context, w gemini.ResponseWriter, r *gemini.Request) {
	fmt.Println("serving page")
	gemini.ServeFile(w, os.DirFS(s.Config.ContentDir), "index.gmi")
}
