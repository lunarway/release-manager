package main

import (
	"flag"
	"log"
	"math/rand"
	"sync"

	amqp "github.com/rabbitmq/amqp091-go"
)

// main is a CLI for publishing messages to a RabbitMQ cluster. It is used to
// test load scenarios against a running RabbitMQ instance/cluster.
//
// Use flag count and size to define how many and how large the messages to
// publish should be.
func main() {
	size := flag.Int("size", 10, "Size of payload in bytes")
	count := flag.Int("count", 10, "Number of messages to publish")
	flag.Parse()

	amqpConn, err := amqp.DialConfig("amqp://lunar:lunar@localhost:5672", amqp.Config{
		Vhost: "/",
	})
	if err != nil {
		panic(err)
	}
	defer amqpConn.Close()

	channel, err := amqpConn.Channel()
	if err != nil {
		panic(err)
	}
	defer channel.Close()
	err = channel.ExchangeDeclare("amqp-load", "topic", true, false, false, false, nil)
	if err != nil {
		panic(err)
	}
	_, err = channel.QueueDeclare("load", true, false, false, false, nil)
	if err != nil {
		panic(err)
	}
	err = channel.QueueBind("load", "#", "amqp-load", false, nil)
	if err != nil {
		panic(err)
	}

	var wg sync.WaitGroup
	wg.Add(*count)
	log.Printf("Publishing %d messages", *count)
	p := payload(*size)
	for i := 0; i < *count; i++ {
		go func() {
			defer wg.Done()
			err = channel.Publish("amqp-load", "load", false, false, amqp.Publishing{
				Body: p,
			})
			if err != nil {
				panic(err)
			}
		}()
	}
	log.Printf("Waiting for publications to complete")
	wg.Wait()
}

func payload(i int) []byte {
	p := make([]byte, i)
	_, err := rand.Read(p)
	if err != nil {
		panic(err)
	}
	return p
}
