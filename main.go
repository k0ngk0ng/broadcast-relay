package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

var (
	version   = "dev"
	buildTime = "unknown"
)

type Config struct {
	ListenPort    int
	ListenAddr    string
	TargetAddrs   []string
	BufferSize    int
	Verbose       bool
	ShowVersion   bool
}

type Relay struct {
	config      *Config
	conn        *net.UDPConn
	targetConns []*net.UDPAddr
	stats       *Stats
	stopChan    chan struct{}
	wg          sync.WaitGroup
}

type Stats struct {
	PacketsReceived uint64
	PacketsForwarded uint64
	BytesReceived   uint64
	BytesForwarded  uint64
	Errors          uint64
	mu              sync.RWMutex
}

func (s *Stats) AddReceived(bytes int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.PacketsReceived++
	s.BytesReceived += uint64(bytes)
}

func (s *Stats) AddForwarded(bytes int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.PacketsForwarded++
	s.BytesForwarded += uint64(bytes)
}

func (s *Stats) AddError() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Errors++
}

func (s *Stats) String() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return fmt.Sprintf("Received: %d packets (%d bytes), Forwarded: %d packets (%d bytes), Errors: %d",
		s.PacketsReceived, s.BytesReceived, s.PacketsForwarded, s.BytesForwarded, s.Errors)
}

func parseConfig() *Config {
	config := &Config{}

	flag.IntVar(&config.ListenPort, "port", 9999, "UDP port to listen for broadcast packets")
	flag.StringVar(&config.ListenAddr, "listen", "0.0.0.0", "Address to listen on (use 0.0.0.0 for all interfaces)")

	var targets string
	flag.StringVar(&targets, "targets", "", "Comma-separated list of target addresses (ip:port), e.g., 192.168.1.100:9999,10.0.0.50:8888")

	flag.IntVar(&config.BufferSize, "buffer", 65535, "UDP buffer size in bytes")
	flag.BoolVar(&config.Verbose, "verbose", false, "Enable verbose logging")
	flag.BoolVar(&config.ShowVersion, "version", false, "Show version information")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Broadcast Relay - Forward local broadcast packets to specified IP:Port\n\n")
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s -port 9999 -targets 192.168.1.100:9999\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -port 9999 -targets 192.168.1.100:9999,10.0.0.50:8888 -verbose\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -listen 0.0.0.0 -port 12345 -targets 192.168.2.1:12345\n", os.Args[0])
	}

	flag.Parse()

	if config.ShowVersion {
		fmt.Printf("Broadcast Relay v%s (built: %s)\n", version, buildTime)
		os.Exit(0)
	}

	if targets == "" {
		fmt.Fprintln(os.Stderr, "Error: -targets is required")
		flag.Usage()
		os.Exit(1)
	}

	// Parse target addresses
	for _, target := range strings.Split(targets, ",") {
		target = strings.TrimSpace(target)
		if target != "" {
			config.TargetAddrs = append(config.TargetAddrs, target)
		}
	}

	if len(config.TargetAddrs) == 0 {
		fmt.Fprintln(os.Stderr, "Error: at least one valid target address is required")
		flag.Usage()
		os.Exit(1)
	}

	return config
}

func NewRelay(config *Config) (*Relay, error) {
	relay := &Relay{
		config:   config,
		stats:    &Stats{},
		stopChan: make(chan struct{}),
	}

	// Resolve target addresses
	for _, target := range config.TargetAddrs {
		addr, err := net.ResolveUDPAddr("udp", target)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve target address %s: %v", target, err)
		}
		relay.targetConns = append(relay.targetConns, addr)
	}

	// Create listening socket
	listenAddr := fmt.Sprintf("%s:%d", config.ListenAddr, config.ListenPort)
	addr, err := net.ResolveUDPAddr("udp", listenAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve listen address: %v", err)
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to create UDP socket: %v", err)
	}

	// Set socket options for receiving broadcast
	if err := conn.SetReadBuffer(config.BufferSize); err != nil {
		log.Printf("Warning: failed to set read buffer size: %v", err)
	}

	relay.conn = conn

	return relay, nil
}

func (r *Relay) Start() {
	log.Printf("Starting Broadcast Relay v%s", version)
	log.Printf("Listening on %s:%d", r.config.ListenAddr, r.config.ListenPort)
	log.Printf("Forwarding to: %v", r.config.TargetAddrs)

	r.wg.Add(1)
	go r.receiveLoop()

	// Start stats reporter if verbose
	if r.config.Verbose {
		r.wg.Add(1)
		go r.statsReporter()
	}
}

func (r *Relay) receiveLoop() {
	defer r.wg.Done()

	buffer := make([]byte, r.config.BufferSize)

	for {
		select {
		case <-r.stopChan:
			return
		default:
		}

		// Set read deadline to allow checking stop channel
		r.conn.SetReadDeadline(time.Now().Add(1 * time.Second))

		n, srcAddr, err := r.conn.ReadFromUDP(buffer)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			select {
			case <-r.stopChan:
				return
			default:
				log.Printf("Error reading UDP packet: %v", err)
				r.stats.AddError()
				continue
			}
		}

		r.stats.AddReceived(n)

		if r.config.Verbose {
			log.Printf("Received %d bytes from %s", n, srcAddr.String())
		}

		// Forward to all targets
		data := buffer[:n]
		for _, target := range r.targetConns {
			// Skip if target is the source (avoid loops)
			if srcAddr.IP.Equal(target.IP) && srcAddr.Port == target.Port {
				if r.config.Verbose {
					log.Printf("Skipping forward to source: %s", target.String())
				}
				continue
			}

			go r.forwardPacket(data, target)
		}
	}
}

func (r *Relay) forwardPacket(data []byte, target *net.UDPAddr) {
	conn, err := net.DialUDP("udp", nil, target)
	if err != nil {
		log.Printf("Error connecting to target %s: %v", target.String(), err)
		r.stats.AddError()
		return
	}
	defer conn.Close()

	n, err := conn.Write(data)
	if err != nil {
		log.Printf("Error forwarding to %s: %v", target.String(), err)
		r.stats.AddError()
		return
	}

	r.stats.AddForwarded(n)

	if r.config.Verbose {
		log.Printf("Forwarded %d bytes to %s", n, target.String())
	}
}

func (r *Relay) statsReporter() {
	defer r.wg.Done()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.stopChan:
			return
		case <-ticker.C:
			log.Printf("Stats: %s", r.stats.String())
		}
	}
}

func (r *Relay) Stop() {
	log.Println("Stopping relay...")
	close(r.stopChan)
	r.conn.Close()
	r.wg.Wait()
	log.Printf("Final stats: %s", r.stats.String())
	log.Println("Relay stopped")
}

func main() {
	config := parseConfig()

	relay, err := NewRelay(config)
	if err != nil {
		log.Fatalf("Failed to create relay: %v", err)
	}

	relay.Start()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	relay.Stop()
}
