package main

import (
	"fmt"
	amqp "github.com/rabbitmq/amqp091-go"
	"listener/event"
	"log"
	"math"
	"os"
	"time"
)

func main() {

	rabbitConn, err := connect()
	if err != nil {
		log.Println("Failed to connect to RabbitMQ", err)
		os.Exit(1)
	}
	defer func(rabbitConn *amqp.Connection) {
		err = rabbitConn.Close()
		if err != nil {
			log.Println("Failed to close RabbitMQ connection", err)
		}
	}(rabbitConn)

	log.Println("Listening for and consuming RabbitMQ messages")

	consumer, err := event.NewConsumer(rabbitConn)
	if err != nil {
		log.Panic("Failed to create consumer", err)
	}

	err = consumer.Listen([]string{"log.INFO", "log.ERROR", "log.WARNING"})
	if err != nil {
		log.Println("Failed to listen for messages", err)
	}

}

func connect() (*amqp.Connection, error) {
	var counts int64
	var backoff = 1 * time.Second
	var connection *amqp.Connection

	for {
		c, err := amqp.Dial("amqp://guest:guest@rabbitmq")
		if err != nil {
			fmt.Println("Failed to connect to RabbitMQ. Retrying in", backoff)
			counts++
		} else {
			log.Println("Connected to RabbitMQ")
			connection = c
			break
		}

		if counts > 5 {
			fmt.Println("Failed to connect to RabbitMQ after 5 retries", err)
			return nil, err
		}

		backoff = time.Duration(math.Pow(float64(counts), 2)) * time.Second
		log.Println("Retrying in", backoff)
		time.Sleep(backoff)
	}

	return connection, nil
}
