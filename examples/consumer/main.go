package main

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/rimdesk/rimnats-go"
	v1 "github.com/rimdesk/rimnats-go/gen/rimdesk/rimnats/v1"
	"google.golang.org/protobuf/proto"
)

var (
	client = rimnats.New("nats://localhost:4333")
)

func init() {
	client.Connect()
}

func main() {
	// Initialize the event bus
	defer client.Close()

	ctx := context.Background()

	// Subscribe to the "product.created" event
	err := client.Subscribe(ctx, "sample.created", "product_stream", "product_service",
		func() proto.Message {
			return &v1.Event{} // Factory method to create a specific event type
		}, func(ctx context.Context, msg proto.Message, m jetstream.Msg) error {
			log.Println("🔥 event received via subject:", m.Subject())

			// Type asserts the message to a specific event type
			event, ok := msg.(*v1.Event)
			if !ok {
				log.Printf("👻 Received an unknown message type")
				return errors.New("unknown message type")
			}

			// Handle the event
			log.Printf("🔥 Event Created: %v", event.String())

			if err := m.Ack(); err != nil {
				log.Println("🚨 Failed to acknowledge message:", err)
			}

			return nil
		})

	if err != nil {
		log.Fatalf("🚨 Failed to subscribe to sample.created event: %v", err)
	}

	// Keep the main function running to receive events
	log.Println(fmt.Sprintf("🚀 waiting for events on ..."))
	select {}
}
