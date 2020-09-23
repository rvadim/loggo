package main

import (
	"encoding/json"
	"log"
	"time"

	"gopkg.in/alecthomas/kingpin.v2"
	"rvadim/loggo/pkg/transport/amqpclient"
	"rvadim/loggo/pkg/transport/redisclient"
)

type config struct {
	transport          string
	createRabbitQueues bool
	amqpURL            string
	amqpExchange       string
	amqpRoutingKey     string
	redisURL           string
	redisKey           string
}

func main() {
	c := config{}
	kingpin.Flag("transport", "Transport type for log messages [amqp | redis]").
		Default("amqp").
		Envar("TRANSPORT").
		StringVar(&c.transport)
	kingpin.Flag("create-rabbit-queues", "Create rabbit queues for testing").
		Envar("CREATE_RABBIT_QUEUES").
		BoolVar(&c.createRabbitQueues)
	kingpin.Flag("amqp-url", "Where to send log messages").
		Default("amqp://localhost/").
		Envar("AMQP_URL").
		StringVar(&c.amqpURL)
	kingpin.Flag("amqp-exchange", "AMQP Exchange for log message delivery").
		Default("amq.direct").
		Envar("AMQP_EXCHANGE").
		StringVar(&c.amqpExchange)
	kingpin.Flag("amqp-routing-key", "AMQP routing key for message delivery").
		Default("all-other").
		Envar("AMQP_ROUTING_KEY").
		StringVar(&c.amqpRoutingKey)
	kingpin.Flag("redis-url", "Where to send log messages").
		Default("localhost:6379").
		Envar("REDIS_URL").
		StringVar(&c.redisURL)
	kingpin.Flag("redis-key", "Redis key for message delivery").
		Default("logs").
		Envar("REDIS_KEY").
		StringVar(&c.redisKey)
	kingpin.Parse()
	if c.transport == "amqp" {
		if c.createRabbitQueues {
			createRabbitQueues(c, 30, 1)
			return
		} else {
			runTests(c)
		}
	}
	if c.transport == "redis" {
		runRedisTests(c)
	}
}

func createRabbitQueues(c config, tries int, timeout int) {
	var broker *amqpclient.Broker
	var err error
	for i := 0; i < tries; i++ {
		broker, err = amqpclient.New(c.amqpURL, c.amqpExchange, c.amqpRoutingKey)
		if err != nil {
			log.Printf("Try #%d, Unable to init amqp client. %s, retry after timeout %d", i, err, timeout)
			time.Sleep(time.Duration(timeout) * time.Second)
			continue
		}
		log.Println("Connection to amqp initialized.")
		break
	}
	defer broker.Close()
	broker.CreateExchange()
	broker.CreateQueue("test", true)
	broker.BindQueue("test")
}

type nginxLog struct {
	TimeMSec  string `json:"time_msec"`
	RequestID string `json:"request_id"`
}

func getExpectedLog() []nginxLog {
	var expectedLog []nginxLog
	expectedLog = append(expectedLog, nginxLog{TimeMSec: "1515474468.380", RequestID: "e7bfe08ae0d17cf3d8ba83fea81a672b"})
	expectedLog = append(expectedLog, nginxLog{TimeMSec: "1515474468.381", RequestID: "551c1c38c45f5086260a5ba919cab365"})
	expectedLog = append(expectedLog, nginxLog{TimeMSec: "1515474468.396", RequestID: "4a1038746802c61a1de10c8802b850c8"})
	expectedLog = append(expectedLog, nginxLog{TimeMSec: "1515474468.396", RequestID: "d010f6631c81407596a7ff19aaf6312b"})
	expectedLog = append(expectedLog, nginxLog{TimeMSec: "1515474468.396", RequestID: "bb994be4fd534ffc4e9605b70c18ee9c"})
	expectedLog = append(expectedLog, nginxLog{TimeMSec: "1515474468.396", RequestID: "a9a150d1c4ab46b15be72da07f6ae35a"})
	expectedLog = append(expectedLog, nginxLog{TimeMSec: "1515474468.757", RequestID: "9ae268549cac561a97678a69431868c8"})
	expectedLog = append(expectedLog, nginxLog{TimeMSec: "1515474468.821", RequestID: "9ae268549cac561a97678a69431868c8"})
	expectedLog = append(expectedLog, nginxLog{TimeMSec: "1515474468.942", RequestID: "d44ee9cad42d204e9dad7acff8cc5466"})
	expectedLog = append(expectedLog, nginxLog{TimeMSec: "1515474469.061", RequestID: "5edc74969d170dd61eaf1cfa90f4ebad"})
	expectedLog = append(expectedLog, nginxLog{TimeMSec: "1515474469.526", RequestID: "b6f12e8ded88b2fa6dc598f5fca1c978"})
	expectedLog = append(expectedLog, nginxLog{TimeMSec: "1515474469.561", RequestID: "b6f12e8ded88b2fa6dc598f5fca1c978"})
	expectedLog = append(expectedLog, nginxLog{TimeMSec: "1515474469.761", RequestID: "85c9dc2c93541d5163b4092ee8a449fb"})
	expectedLog = append(expectedLog, nginxLog{TimeMSec: "1515474469.812", RequestID: "285e18e863aa016bb975f08d0761abc6"})
	expectedLog = append(expectedLog, nginxLog{TimeMSec: "1515474470.067", RequestID: "23b2aa35cc21972124e9801d6657d4b6"})
	expectedLog = append(expectedLog, nginxLog{TimeMSec: "1515474470.215", RequestID: "9ea29074902088fc11f4f21065abf177"})
	expectedLog = append(expectedLog, nginxLog{TimeMSec: "1515474470.251", RequestID: "1216ed217cb8c6c0d76d0da311c91b26"})
	return expectedLog
}

func isInExpeced(actualLog *nginxLog, expected []nginxLog) bool {
	for _, elog := range expected {
		if elog.TimeMSec == actualLog.TimeMSec && elog.RequestID == actualLog.RequestID {
			return true
		}
	}
	log.Printf("Warning! Unable to find actual log message '%v' in expected array", actualLog)
	return false
}

func runTests(c config) {
	broker, err := amqpclient.New(c.amqpURL, c.amqpExchange, c.amqpRoutingKey)
	if err != nil {
		log.Fatalf("Unable to init amqp client. %s", err)
	}
	defer broker.Close()

	ch, err := broker.Consume("test")
	if err != nil {
		log.Fatalf("Consume failed: %s", err)
	}
	var i = 0
	var matched = 0

	expected := getExpectedLog()
	log.Println("Wait for messages in channel")
	for message := range ch {
		l := &nginxLog{}
		err := json.Unmarshal(message.Body, l)
		if err != nil {
			log.Printf("Unable to parse %s, %s", message.Body, err)
		}
		if isInExpeced(l, expected) {
			matched++
		}
		if i >= 16 {
			break
		}
		i++
	}

	if matched != 17 {
		log.Fatalf("Test failed, not all log files matched")
	} else {
		log.Println("Tests successful")
	}
}

func runRedisTests(c config) {
	client, err := redisclient.New(c.redisURL, c.redisKey)
	if err != nil {
		log.Fatalf("Unable to init redis client. %s", err)
	}
	defer client.Close()

	var matched = 0

	expected := getExpectedLog()
	log.Println("Wait for messages in channel")
	for i := 0; i < 17; i++ {
		message, err := client.ReceiveMessage()
		if err != nil {
			log.Fatalf("Receive failed: %s", err)
		}

		l := &nginxLog{}
		err = json.Unmarshal(message, l)
		if err != nil {
			log.Printf("Unable to parse %s, %s", message, err)
		}
		if isInExpeced(l, expected) {
			matched++
		}
	}

	if matched != 17 {
		log.Fatalf("Test failed, not all log files matched")
	} else {
		log.Println("Tests successful")
	}
}
