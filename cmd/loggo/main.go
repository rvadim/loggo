package main

import (
	"log"
	"os"
	"os/signal"
	"rvadim/loggo/pkg/transport/firehose"
	"syscall"

	"rvadim/loggo/pkg/config"
	"rvadim/loggo/pkg/docker"
	"rvadim/loggo/pkg/service"
	"rvadim/loggo/pkg/storage"
	"rvadim/loggo/pkg/transport"
	"rvadim/loggo/pkg/transport/amqpclient"
	"rvadim/loggo/pkg/transport/redisclient"
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile) // v.reyder: Log also file name and line number
	c := config.GetConfig()
	log.Printf("Starting with configuration: %s", c.ToString())

	var broker transport.ITransportClient
	var err error

	if c.Transport == "amqp" {
		broker, err = amqpclient.New(c.AMQPURL, c.AMQPExchange, c.AMQPRoutingKey)
		if err != nil {
			log.Fatalf("Unable to init amqp client. %s", err)
		}
	} else if c.Transport == "firehose" {
		broker, err = firehose.New(c.FireHoseDeliveryStream)
		if err != nil {
			log.Fatalf("Unable to init firehose client, %s", err)
		}
	} else {
		broker, err = redisclient.New(c.RedisURL, c.RedisKey, c.RedisPassword)
		if err != nil {
			log.Fatalf("Unable to init redis client. %s", err)
		}
	}

	registry, err := storage.NewRegistryFile(c.PositionFilePath, 1)
	if err != nil {
		log.Fatalln(err)
	}
	defer registry.Close()

	finder, err := docker.NewFinder(c.LogsPath)
	if err != nil {
		log.Fatalln(err)
	}

	s := service.NewService(c, registry, broker, finder)

	go s.Start()

	// Handle SIGINT and SIGTERM.
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	log.Printf("Catched signal '%s'", <-ch)

	// Stop the service gracefully.
	s.Stop()
	log.Println("Loggo successfully stopped now.")
}
