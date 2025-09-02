package main

import (
	"context"
	"log"
	"time"

	"github.com/rimdesk/rimnats-go"
	v1 "github.com/rimdesk/rimnats-go/gen/rimdesk/rimnats/v1"
	"google.golang.org/protobuf/proto"
)

var (
	client = rimnats.New("nats://localhost:4222")
)

func init() {
	client.Connect()
}

func main() {
	defer client.Close()

	ctx := context.Background()
	resp, err := client.Request(ctx, "example.say.hello", &v1.SayHelloRequest{Name: "Joey"}, func() proto.Message { return &v1.SayHelloResponse{} }, 3*time.Second)
	if err != nil {
		log.Println("🚨 response error:", err)
	}

	response := resp.(*v1.SayHelloResponse)

	log.Println("==== 📣 Response==== :", response.GetMessage())
}
