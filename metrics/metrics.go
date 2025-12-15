package metrics

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// Metrics holds the bot's runtime metrics
type Metrics struct {
	StartTime        time.Time
	MessagesReceived atomic.Uint64
	MessagesSent     atomic.Uint64
	CommandsExecuted atomic.Uint64
	Reconnects       atomic.Uint64
	Connected        atomic.Bool
	mu               sync.RWMutex
	channels         map[string]bool
}

var globalMetrics *Metrics

// Init initializes the global metrics
func Init() *Metrics {
	globalMetrics = &Metrics{
		StartTime: time.Now(),
		channels:  make(map[string]bool),
	}
	return globalMetrics
}

// Get returns the global metrics instance
func Get() *Metrics {
	return globalMetrics
}

// IncMessagesReceived increments the messages received counter
func (m *Metrics) IncMessagesReceived() {
	m.MessagesReceived.Add(1)
}

// IncMessagesSent increments the messages sent counter
func (m *Metrics) IncMessagesSent() {
	m.MessagesSent.Add(1)
}

// IncCommandsExecuted increments the commands executed counter
func (m *Metrics) IncCommandsExecuted() {
	m.CommandsExecuted.Add(1)
}

// IncReconnects increments the reconnection counter
func (m *Metrics) IncReconnects() {
	m.Reconnects.Add(1)
}

// SetConnected sets the connection status
func (m *Metrics) SetConnected(connected bool) {
	m.Connected.Store(connected)
}

// AddChannel marks a channel as joined
func (m *Metrics) AddChannel(channel string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.channels[channel] = true
}

// RemoveChannel marks a channel as left
func (m *Metrics) RemoveChannel(channel string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.channels, channel)
}

// GetChannelCount returns the number of channels the bot is in
func (m *Metrics) GetChannelCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.channels)
}

// Server implements a Prometheus-compatible metrics HTTP server
type Server struct {
	port string
	srv  *http.Server
}

// NewServer creates a new metrics server
func NewServer(port string) *Server {
	return &Server{
		port: port,
	}
}

// Start begins serving metrics on the configured port
func (s *Server) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/metrics", s.metricsHandler)
	mux.HandleFunc("/health", s.healthHandler)

	s.srv = &http.Server{
		Addr:    ":" + s.port,
		Handler: mux,
	}

	log.Printf("Metrics server listening on port %s", s.port)

	go func() {
		if err := s.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Metrics server error: %v", err)
		}
	}()

	return nil
}

// Stop stops the metrics server
func (s *Server) Stop() error {
	if s.srv != nil {
		return s.srv.Close()
	}
	return nil
}

func (s *Server) metricsHandler(w http.ResponseWriter, r *http.Request) {
	m := Get()
	if m == nil {
		http.Error(w, "Metrics not initialized", http.StatusInternalServerError)
		return
	}

	uptime := time.Since(m.StartTime).Seconds()
	connected := 0
	if m.Connected.Load() {
		connected = 1
	}

	// Prometheus text format
	fmt.Fprintf(w, "# HELP marmithon_uptime_seconds Bot uptime in seconds\n")
	fmt.Fprintf(w, "# TYPE marmithon_uptime_seconds gauge\n")
	fmt.Fprintf(w, "marmithon_uptime_seconds %.2f\n", uptime)
	fmt.Fprintf(w, "\n")

	fmt.Fprintf(w, "# HELP marmithon_connected Connection status (1=connected, 0=disconnected)\n")
	fmt.Fprintf(w, "# TYPE marmithon_connected gauge\n")
	fmt.Fprintf(w, "marmithon_connected %d\n", connected)
	fmt.Fprintf(w, "\n")

	fmt.Fprintf(w, "# HELP marmithon_messages_received_total Total messages received\n")
	fmt.Fprintf(w, "# TYPE marmithon_messages_received_total counter\n")
	fmt.Fprintf(w, "marmithon_messages_received_total %d\n", m.MessagesReceived.Load())
	fmt.Fprintf(w, "\n")

	fmt.Fprintf(w, "# HELP marmithon_messages_sent_total Total messages sent\n")
	fmt.Fprintf(w, "# TYPE marmithon_messages_sent_total counter\n")
	fmt.Fprintf(w, "marmithon_messages_sent_total %d\n", m.MessagesSent.Load())
	fmt.Fprintf(w, "\n")

	fmt.Fprintf(w, "# HELP marmithon_commands_executed_total Total commands executed\n")
	fmt.Fprintf(w, "# TYPE marmithon_commands_executed_total counter\n")
	fmt.Fprintf(w, "marmithon_commands_executed_total %d\n", m.CommandsExecuted.Load())
	fmt.Fprintf(w, "\n")

	fmt.Fprintf(w, "# HELP marmithon_reconnects_total Total reconnection attempts\n")
	fmt.Fprintf(w, "# TYPE marmithon_reconnects_total counter\n")
	fmt.Fprintf(w, "marmithon_reconnects_total %d\n", m.Reconnects.Load())
	fmt.Fprintf(w, "\n")

	fmt.Fprintf(w, "# HELP marmithon_channels Number of channels joined\n")
	fmt.Fprintf(w, "# TYPE marmithon_channels gauge\n")
	fmt.Fprintf(w, "marmithon_channels %d\n", m.GetChannelCount())
}

func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	m := Get()
	if m == nil || !m.Connected.Load() {
		http.Error(w, "Not connected", http.StatusServiceUnavailable)
		return
	}
	fmt.Fprintf(w, "OK\n")
}
