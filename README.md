# Nexor
A shooter package for handling events with nats jetstream

### Publishing events
An example of publishing an event is shown below:

```go
package main

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/rimdesk/rimnats"
	v1 "github.com/rimdesk/rimnats/gen/shooters/nexor/v1"
	"log"
	"strings"
	"time"
)

var (
	client = nexor.New("nats://localhost:4222")
)

func init() {
	client.Connect()
}

func main() {
	// Initialize the event bus
	eventBus := client.GetEngine()
	js := eventBus.JetStream()
	defer eventBus.Close()

	// Create a stream (e.g. product.*)
	_, err := js.AddStream(&nats.StreamConfig{
		Name:     "PRODUCT_EVENTS",
		Subjects: []string{"product.*"},
		Storage:  nats.FileStorage,
	})
	if err != nil && !strings.Contains(err.Error(), "stream name already in use") {
		log.Fatalf("❌ Failed to create stream: %v", err)
	}

	for {
		// Get the current time
		currentTime := time.Now()
		// List of sample words
		words := []string{"Apple", "Banana", "Orange", "Mango", "Grape", "Peach", "Plum", "Cherry", "Lemon", "Lime"}
		// Generate a random word
		randomWord := words[currentTime.UnixNano()%int64(len(words))]

		// Create a ProductCreated event
		event := &v1.ProductCreated{
			Id:         uuid.NewString(),
			Name:       randomWord,
			SupplierId: uuid.NewString(),
			CreatedAt:  currentTime.UnixMilli(),
		}

		// Publish the event
		if err := eventBus.Publish(context.Background(), "product.created", event); err != nil {
			log.Fatalf("Failed to publish event: %v", err)
		}

		fmt.Println("🚀 Event published successfully! 🚀")

		time.Sleep(time.Duration(5) * time.Second)
	}
}

```

### Subscribing to events
An example of subscribing to events is shown below:

```go
package main

import (
	"context"
	"errors"
	"github.com/nats-io/nats.go"
	"github.com/rimdesk/rimnats"
	v1 "github.com/rimdesk/rimnats/gen/shooters/nexor/v1"
	"google.golang.org/protobuf/proto"
	"log"
)

var (
	client = nexor.New("nats://localhost:4222")
)

func init() {
	client.Connect()
}

func main() {
	// Initialize the event bus
	eventBus := client.GetEngine()
	defer eventBus.Close()

	// Subscribe to the "product.created" event
	err := eventBus.Subscribe("product.created", "product_created_consumer", func() proto.Message {
		return &v1.ProductCreated{} // Factory method to create a specific event type
	}, func(ctx context.Context, msg proto.Message, m *nats.Msg) error {
		log.Println("🔥 event received via subject:", m.Subject)

		// Type asserts the message to a specific event type
		productCreatedEvent, ok := msg.(*v1.ProductCreated)
		if !ok {
			log.Printf("👻 Received an unknown message type")
			return errors.New("unknown message type")
		}

		// Handle the event
		log.Printf("🔥 Product Created: %v", productCreatedEvent.String())

		if err := m.Ack(); err != nil {
			log.Println("❌ Failed to acknowledge message:", err)
		}

		return nil
	})

	if err != nil {
		log.Fatalf("❌ Failed to subscribe to product.created event: %v", err)
	}

	// Keep the main function running to receive events
	log.Println("🚀 waiting for events...")
	select {}
}
```

### Environment variables:
The default parameters can be overridden by setting the following environment variables:

```text
NEXOR.CLIENT=Nexor
NEXOR.DEBUG=false
NEXOR.URL=nats://localhost:4222
NEXOR.MAX_CONNECTIONS=5
NEXOR.MAX_RECONNECT_WAIT=5
```