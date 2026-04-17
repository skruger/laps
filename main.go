package main

import (
	"context"
	"fmt"
	"laps/config"
	"laps/server"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

func main() {
	if len(os.Args) < 2 {
		run_server()
	}
	if os.Args[1] == "client" {
		run_client()
	} else if os.Args[1] == "server" {
		run_server()
	} else {
		help()
	}
}

func help() {
	fmt.Fprintf(os.Stderr, "Usage: %s client|server\n", os.Args[0])
	os.Exit(1)
}

func run_server() {
	// determine config file from environment or default
	cfgFile := os.Getenv("LAPS_CONFIG")
	if cfgFile == "" {
		cfgFile = "/etc/laps/config.yaml"
	}

	cfg, err := config.LoadConfig(cfgFile)
	if err != nil {
		log.Fatalf("error: failed to load config %s: %v; using defaults", cfgFile, err)
	}

	var wg sync.WaitGroup

	server_ctx, server_stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer server_stop()

	wg.Go(func() {
		srv := server.NewLapsServer(cfg)
		srv.Run(server_ctx)
	})

	mdns_ctx, mdns_stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer mdns_stop()
	wg.Go(func() {
		server.StartMDNS(mdns_ctx, cfg)
	})
	wg.Wait()

}

func run_client() {
	log.Fatal("Client mode not implemented yet")
}
