package tr069

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

// Server is a minimal ACS that answers Inform messages and pushes one RPC.
type Server struct {
	Addr         string // e.g. ":7547"
	Path         string // e.g. "/acs"
	DownloadURL  string
	DownloadSize int64
	RPC          string // set-params | download | reboot | none
	ParamName    string
	ParamValue   string
	ServeFile    string // optional local file exposed at /config.bin
	DryRun       bool

	srv          *http.Server
	mu           sync.Mutex
	informSeen   bool
	rpcPushed    bool
	requestCount int
}

// Start starts the ACS HTTP server and blocks until it stops.
func (s *Server) Start() error {
	mux := http.NewServeMux()
	mux.Handle(s.Path, s)
	mux.HandleFunc("/config.bin", s.serveConfigFile)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		s.writeInstructions(w)
	})

	s.srv = &http.Server{
		Addr:              s.Addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}
	log.Infof("[acs] listening on http://%s%s", s.Addr, s.Path)
	return s.srv.ListenAndServe()
}

// Shutdown gracefully stops the server.
func (s *Server) Shutdown(ctx context.Context) error {
	if s.srv == nil {
		return nil
	}
	return s.srv.Shutdown(ctx)
}

// ServeHTTP implements the CWMP conversation.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeInstructions(w)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 8<<20))
	if err != nil {
		http.Error(w, "read error", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/xml; charset=utf-8")

	s.mu.Lock()
	if !s.informSeen {
		s.informSeen = true
		s.requestCount++
		s.mu.Unlock()
		log.Infof("[acs] received Inform (%d bytes); replying with InformResponse", len(body))
		fmt.Fprint(w, Envelope(Header("1"), InformResponse()))
		return
	}
	s.requestCount++
	// The RPC is delivered once per session; further CPE posts get an empty
	// envelope so a single Inform does not trigger a flood of commands.
	push := !s.rpcPushed
	if push {
		s.rpcPushed = true
	}
	s.mu.Unlock()

	log.Debugf("[acs] CPE request #%d (%d bytes)", s.requestCount, len(body))
	if push {
		fmt.Fprint(w, s.nextRPC())
	} else {
		fmt.Fprint(w, Envelope(Header("1"), ""))
	}
}

func (s *Server) nextRPC() string {
	if s.DryRun {
		log.Warn("[acs] dry-run: suppressing RPC push")
		return Envelope(Header("1"), "")
	}

	switch s.RPC {
	case "download":
		if s.DownloadURL == "" {
			log.Warn("[acs] download requested but no URL configured")
			return Envelope(Header("1"), "")
		}
		log.Infof("[acs] pushing Download RPC -> %s", s.DownloadURL)
		return Envelope(Header("1"), Download(s.DownloadURL, "", "", s.DownloadSize))
	case "reboot":
		log.Info("[acs] pushing Reboot RPC")
		return Envelope(Header("1"), Reboot())
	case "set-params", "":
		name := s.ParamName
		if name == "" {
			name = "InternetGatewayDevice.ManagementServer.URL"
		}
		log.Infof("[acs] pushing SetParameterValues %s=%s", name, s.ParamValue)
		return Envelope(Header("1"), SetParameterValues([]Parameter{{Name: name, Value: s.ParamValue}}, "zlt-unlocker"))
	case "none":
		return Envelope(Header("1"), "")
	default:
		log.Warnf("[acs] unknown RPC %q; sending empty envelope", s.RPC)
		return Envelope(Header("1"), "")
	}
}

func (s *Server) serveConfigFile(w http.ResponseWriter, r *http.Request) {
	if s.ServeFile == "" {
		http.NotFound(w, r)
		return
	}
	log.Infof("[acs] serving %s", s.ServeFile)
	http.ServeFile(w, r, s.ServeFile)
}

func (s *Server) writeInstructions(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintln(w, "zlt-unlocker ACS simulator is running.")
	fmt.Fprintf(w, "Set the router's ACS URL to: http://<this-host>%s\n", s.Path)
}
