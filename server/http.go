package server

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"laps/config"
	"laps/dnsclient"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// LapsServer holds state for a small HTTP server used by the example.
type LapsServer struct {
	Addr string
	srv  *http.Server
	cfg  *config.Config
}

// NewLapsServer returns a configured LapsServer with sensible defaults.
func NewLapsServer(cfg *config.Config) *LapsServer {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, "Hello World")
	})
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, "OK")
	})
	mux.HandleFunc("/update_dns", dnsUpdateHandler(cfg))

	s := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", cfg.ListenAddr, cfg.ListenPort),
		Handler: mux,
	}

	return &LapsServer{
		Addr: s.Addr,
		srv:  s,
		cfg:  cfg,
	}
}

// Run starts the HTTP server and blocks until the process is interrupted
// (SIGINT/SIGTERM) or the server exits with an error.
func (f *LapsServer) Run(parentCtx context.Context) {
	// create a context that cancels on SIGINT/SIGTERM
	ctx, stop := signal.NotifyContext(parentCtx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)

	go func() {
		log.Printf("starting server on %s", f.Addr)
		if err := f.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		} else {
			errCh <- nil
		}
	}()

	select {
	case <-ctx.Done():
		// graceful shutdown with timeout
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		log.Println("shutting down server...")
		if err := f.srv.Shutdown(shutdownCtx); err != nil {
			log.Printf("server shutdown error: %v", err)
		}
		return
	case err := <-errCh:
		if err != nil {
			log.Fatalf("server error: %v", err)
		}
		return
	}
}

type dnsUpdateRequest struct {
	Hostname  string `json:"hostname"`
	IPv6Addr  string `json:"ipv6_addr"`
	IPv4Addr  string `json:"ipv4_addr"`
	Timestamp int64  `json:"timestamp"`
	Signature string `json:"signature"`
}

func (r *dnsUpdateRequest) CheckSignature(psk string) bool {
	minTime := time.Now().Unix() - 30
	maxTime := minTime + 60
	if !(minTime < r.Timestamp && r.Timestamp < maxTime) {
		log.Printf("Signature time out of range: %d not in [%d, %d]", r.Timestamp, minTime, maxTime)
		return false
	}

	raw := fmt.Sprintf("%s|%s|%s|%d|%s", r.Hostname, r.IPv6Addr, r.IPv4Addr, r.Timestamp, psk)
	sum := sha256.Sum256([]byte(raw))
	sumStr := hex.EncodeToString(sum[:])
	log.Print(sumStr)
	return fmt.Sprintf("%x", sum) == r.Signature
}

func dnsUpdateHandler(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var req dnsUpdateRequest

		// Ensure body is closed
		defer r.Body.Close()

		dec := json.NewDecoder(r.Body)
		// disallow unknown fields to catch typos
		dec.DisallowUnknownFields()
		if err := dec.Decode(&req); err != nil {
			http.Error(w, "invalid JSON body", http.StatusBadRequest)
			log.Printf("dns update: invalid request body: %v", err)
			return
		}

		// simple validation
		if req.Hostname == "" || req.IPv6Addr == "" || req.Signature == "" {
			http.Error(w, "missing required fields", http.StatusBadRequest)
			return
		}

		// TODO: verify signature using configured clients / PSK
		for _, client := range cfg.Clients {
			if req.Hostname == client.Hostname {
				if req.CheckSignature(client.PSK) {
					// TODO: update route53 DNS
					log.Printf("Updating %s to %s and %s", req.Hostname, req.IPv6Addr, req.IPv4Addr)
					err := dnsclient.UpdateRoute53(context.Background(), cfg, req.Hostname, req.IPv6Addr, req.IPv4Addr)
					if err == nil {
						w.WriteHeader(http.StatusOK)
						_, _ = w.Write([]byte("ok"))
						return
					} else {
						log.Printf("error updating DNS for %s: %s", req.Hostname, err)
					}
				}
			}
		}

		http.Error(w, "no valid signature", http.StatusBadRequest)
	}
}
