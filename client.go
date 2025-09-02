// Package rimnats provides a NATS client implementation with support for JetStream.
// It simplifies the process of connecting to NATS servers and managing connections
// with configurable options through environment variables.
package nexor

import (
	"context"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"google.golang.org/protobuf/proto"
)

type Client interface {
	Close()
	Connect()
	CreateStream(ctx context.Context, config jetstream.StreamConfig) error
	GetEngine() *nexor
	JetStream() jetstream.JetStream
	Publish(ctx context.Context, subject string, msg proto.Message, opts ...jetstream.PublishOpt) error
	Reply(subject string, reqFactory func() proto.Message, handler func(context.Context, proto.Message) (proto.Message, error)) error
	Request(ctx context.Context, subject string, req proto.Message, factory func() proto.Message, timeout time.Duration) (proto.Message, error)
	Subscribe(ctx context.Context, subject, stream, durable string, factory func() proto.Message, handler ProtoHandler, opts ...jetstream.PullConsumeOpt) error
}

// Nexor represents a NATS client with JetStream support.
type nexor struct {
	conn *nats.Conn          // Connection to the NATS server
	js   jetstream.JetStream // JetStream context for pub/sub operations
	cfg  *nexorConfig        // Configuration for the NATS client
}

func (n *nexor) CreateStream(ctx context.Context, config jetstream.StreamConfig) error {
	_, err := n.js.CreateOrUpdateStream(ctx, config)
	if err != nil {
		log.Fatalf("🚨 Failed to create stream: %v", err)
	}

	return nil
}

func (n *nexor) GetEngine() *nexor {
	return n
}

func (n *nexor) Connect() {
	conn, err := nats.Connect(n.cfg.Url, n.cfg.Opts...)
	if err != nil {
		if n.cfg.Debug {
			log.Printf("🔌 Failed to connect to NATS: %v 🔌\n\n", err)
			os.Exit(1)
		}
		os.Exit(1)
	}

	js, err := jetstream.New(conn)
	if err != nil {
		if n.cfg.Debug {
			log.Printf("🔌 Failed to connect to Jetstream: %v 🔌", err)
			os.Exit(1)
		}

		conn.Close()
		os.Exit(1)
	}

	n.conn = conn
	n.js = js

	if n.cfg.Debug {
		log.Printf("🚀 Connected to NATS server successful 🚀\n")
	}
}

// nexorConfig holds the configuration parameters for the NATS client.
type nexorConfig struct {
	Url        string        // Url is the address of the NATS server for client connection.
	ClientName string        // Name of the client used for connection identification
	Debug      bool          // Enable debug mode for verbose logging
	MaxConn    int           // Maximum number of allowed connections
	MaxRecon   int           // Maximum number of reconnection attempts
	ReconWait  int           // Time to wait between reconnection attempts in seconds
	Opts       []nats.Option // Opts specifies additional NATS options for configuring the client connection or behavior.
}

// getConfig retrieves the configuration from environment variables and returns
// a nexorConfig with either default values or those specified in the environment.
func getConfig() *nexorConfig {
	var debugMode = false
	var clientName = "Nexor"
	var maxConn, maxWait = 5, 5
	if debugModeValue, found := os.LookupEnv("NEXOR.DEBUG"); found {
		debugMode = debugModeValue == "true"
	}

	if clientNameValue, found := os.LookupEnv("NEXOR.CLIENT"); found {
		clientName = clientNameValue
	}

	if maxConnValue := os.Getenv("NEXOR.MAX_CONNECTIONS"); maxConnValue != "" {
		maxConn, _ = strconv.Atoi(maxConnValue)
	}

	if maxWaitValue := os.Getenv("NEXOR.MAX_RECONNECT_WAIT"); maxWaitValue != "" {
		maxWait, _ = strconv.Atoi(maxWaitValue)
	}

	return &nexorConfig{
		ClientName: clientName,
		Debug:      debugMode,
		MaxConn:    maxConn,
		ReconWait:  maxWait,
	}
}

// New creates a new Nexor instance connected to the specified NATS server.
// It accepts a URL string and optional NATS options. If no options are provided,
// it uses default configuration values from environment variables.
// Returns a configured Nexor instance and any error encountered during connection.
func New(url string, opts ...nats.Option) Client {
	cfg := getConfig()
	cfg.Url = url
	cfg.Opts = opts

	if len(opts) == 0 {
		opts = []nats.Option{
			nats.Name(cfg.ClientName),
			nats.MaxReconnects(cfg.MaxRecon),
			nats.ReconnectWait(time.Duration(cfg.ReconWait) * time.Second),
		}
	}

	return &nexor{cfg: cfg}
}

// Close safely closes the NATS connection.
func (n *nexor) Close() {
	if n.conn != nil && !n.conn.IsClosed() {
		n.conn.Close()
	}
}

// JetStream exposes the underlying JetStream context
// so that microservices can create/manage streams and consumers.
func (n *nexor) JetStream() jetstream.JetStream {
	return n.js
}
