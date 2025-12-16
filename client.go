// Package rimnats provides a NATS client implementation with support for JetStream.
// It simplifies the process of connecting to NATS servers and managing connections
// with configurable options through environment variables.
package rimnats

import (
	"context"
	"os"
	"strconv"
	"time"

	"github.com/beego/beego/v2/core/logs"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"google.golang.org/protobuf/proto"
)

type Client interface {
	Close()
	Connect()
	GetEngine() *rimNats
	JetStream() jetstream.JetStream
	CreateStream(ctx context.Context, config jetstream.StreamConfig) error
	Publish(ctx context.Context, subject string, msg proto.Message, opts ...jetstream.PublishOpt) error
	Reply(subject string, reqFactory func() proto.Message, handler func(context.Context, proto.Message) (proto.Message, error)) error
	Request(ctx context.Context, subject string, req proto.Message, factory func() proto.Message, timeout time.Duration) (proto.Message, error)
	Subscribe(ctx context.Context, subject, stream, durable string, factory func() proto.Message, handler ProtoHandler, opts ...jetstream.PullConsumeOpt) error
}

// Rimnats represents a NATS client with JetStream support.
type rimNats struct {
	conn  *nats.Conn          // Connection to the NATS server
	cfg   *nexorConfig        // Configuration for the NATS client
	loggR *logs.BeeLogger     // Beego logger for logging
	js    jetstream.JetStream // JetStream context for pub/sub operations
}

func (n *rimNats) CreateStream(ctx context.Context, config jetstream.StreamConfig) error {
	_, err := n.js.CreateOrUpdateStream(ctx, config)
	if err != nil {
		n.loggR.Error("🚨 Failed to create stream: %v", err)
	}

	return nil
}

func (n *rimNats) GetEngine() *rimNats {
	return n
}

func (n *rimNats) Connect() {
	conn, err := nats.Connect(n.cfg.Url, n.cfg.Opts...)
	if err != nil {
		if n.cfg.Debug {
			n.loggR.Error("🔌 Failed to connect to NATS: %v", err)
			os.Exit(1)
		}
		os.Exit(1)
	}

	js, err := jetstream.New(conn)
	if err != nil {
		if n.cfg.Debug {
			n.loggR.Error("🔌 Failed to connect to Jetstream: %v 🔌", err)
			os.Exit(1)
		}

		conn.Close()
		os.Exit(1)
	}

	n.conn = conn
	n.js = js

	if n.cfg.Debug {
		n.loggR.Info("🚀 Connected to NATS server successful")
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
	var clientName = "Rimnats"
	var maxConn, maxWait = 5, 5
	if debugModeValue, found := os.LookupEnv("RIMNATS.DEBUG"); found {
		debugMode = debugModeValue == "true"
	}

	if clientNameValue, found := os.LookupEnv("RIMNATS.CLIENT"); found {
		clientName = clientNameValue
	}

	if maxConnValue := os.Getenv("RIMNATS.MAX_CONNECTIONS"); maxConnValue != "" {
		maxConn, _ = strconv.Atoi(maxConnValue)
	}

	if maxWaitValue := os.Getenv("RIMNATS.MAX_RECONNECT_WAIT"); maxWaitValue != "" {
		maxWait, _ = strconv.Atoi(maxWaitValue)
	}

	return &nexorConfig{
		ClientName: clientName,
		Debug:      debugMode,
		MaxConn:    maxConn,
		ReconWait:  maxWait,
	}
}

func getLogger() *logs.BeeLogger {
	beeLogger := logs.NewLogger(10000)

	beeLogger.SetLogger(
		logs.AdapterConsole,
		`{"level":6,"color":true}`,
	)

	beeLogger.EnableFuncCallDepth(true)
	beeLogger.SetLogFuncCallDepth(3)

	beeLogger.Async(1000)

	return beeLogger
}

// New creates a new Rimnats instance connected to the specified NATS server.
// It accepts a URL string and optional NATS options. If no options are provided,
// it uses default configuration values from environment variables.
// Returns a configured Rimnats instance and any error encountered during connection.
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

	return &rimNats{cfg: cfg, loggR: getLogger()}
}

// Close safely closes the NATS connection.
func (n *rimNats) Close() {
	if n.conn != nil && !n.conn.IsClosed() {
		n.conn.Close()
	}
}

// JetStream exposes the underlying JetStream context
// so that microservices can create/manage streams and consumers.
func (n *rimNats) JetStream() jetstream.JetStream {
	return n.js
}
