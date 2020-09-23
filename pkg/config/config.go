package config

import (
	"fmt"
	"reflect"

	"regexp"

	"log"

	"gopkg.in/alecthomas/kingpin.v2"
)

// Config store all configuration options
type Config struct {
	LogsPath               string
	PositionFilePath       string
	DirRereadIntervalSec   int
	ReaderMaxChunk         int
	ReaderTimeoutSec       int
	AMQPURL                string
	AMQPExchange           string
	AMQPRoutingKey         string
	RedisURL               string
	RedisKey               string
	Transport              string
	DataCenter             string
	Purpose                string
	NodeHostname           string
	LogType                string
	LogstashPrefix         string
	ExcludeRegex           *regexp.Regexp
	IncludeRegex           *regexp.Regexp
	excludeRegex           string
	includeRegex           string
	FireHoseDeliveryStream string
}

// GetConfig generate Config from options and env vars
func GetConfig() *Config {
	c := &Config{}
	kingpin.Flag("transport", "Transport type for log messages [amqp | redis | firehose]").
		Default("amqp").
		Envar("TRANSPORT").
		StringVar(&c.Transport)
	kingpin.Flag("redis-hostname", "Where to send log messages").
		Default("localhost:6379").
		Envar("REDIS_HOSTNAME").
		StringVar(&c.RedisURL)
	kingpin.Flag("redis-key", "Where to send log messages").
		Default("logs").
		Envar("REDIS_KEY").
		StringVar(&c.RedisKey)
	kingpin.Flag("amqp-url", "Where to send log messages").
		Default("amqp://localhost/").
		Envar("AMQP_URL").
		StringVar(&c.AMQPURL)
	kingpin.Flag("amqp-exchange", "AMQP Exchange for log message delivery").
		Default("amq.direct").
		Envar("AMQP_EXCHANGE").
		StringVar(&c.AMQPExchange)
	kingpin.Flag("amqp-routing-key", "AMQP routing key for message delivery").
		Default("all-other").
		Envar("AMQP_ROUTING_KEY").
		StringVar(&c.AMQPRoutingKey)
	kingpin.Flag("firehose-delivery-stream", "AWS FireHose delivery stream, only with transport == 'firehose'").
		Default("my-delivery").
		Envar("FIREHOSE_DELIVERY_STREAM").
		StringVar(&c.FireHoseDeliveryStream)
	kingpin.Flag("logs-path", "Path where loggo will watch for log files").
		Default("/var/log/pods/").
		Envar("LOGS_PATH").
		StringVar(&c.LogsPath)
	kingpin.Flag("position-file-path", "Path to file where loggo store read position").
		Default("/var/log/loggo-logs.pos").
		Envar("POSITION_FILE_PATH").
		StringVar(&c.PositionFilePath)
	kingpin.Flag("logs-path-reread-interval_sec", "How often reread logs-path directory searching for new log files").
		Default("10").
		Envar("LOGS_PATH_REREAD_INTERVAL_SEC").
		IntVar(&c.DirRereadIntervalSec)
	kingpin.Flag("reader-max-chunk", "How long log lines store in buffer, before send to storage").
		Default("1000").
		Envar("READER_MAX_CHUNK").
		IntVar(&c.ReaderMaxChunk)
	kingpin.Flag("reader-timeout-sec", "How long to wait, before start read log file which not add logs last time").
		Default("5").
		Envar("READER_TIMEOUT_SEC").
		IntVar(&c.ReaderTimeoutSec)
	kingpin.Flag("dc", "Current datacenter id, will be included to each log message").
		Default("n3").
		Envar("DC").
		StringVar(&c.DataCenter)
	kingpin.Flag("purpose", "Current datacenter purpose, will be included to each log message").
		Default("staging").
		Envar("PURPOSE").
		StringVar(&c.Purpose)
	kingpin.Flag("node-hostname", "Current node hostname, will be included to each log message").
		Default("localhost").
		Envar("NODE_HOSTNAME").
		StringVar(&c.NodeHostname)
	kingpin.Flag("log-type", "Current log type, will be included to each log message").
		Default("containers").
		Envar("LOG_TYPE").
		StringVar(&c.LogType)
	kingpin.Flag("logstash-prefix", "Current logstash prefix, will be included to each log message").
		Default("k8s-unknown").
		Envar("LOGSTASH_PREFIX").
		StringVar(&c.LogstashPrefix)
	kingpin.Flag("exclude-regex", "Which files exclude from processing, can't be used with include-regex").
		Default("").
		Envar("EXCLUDE_REGEX").
		StringVar(&c.excludeRegex)
	kingpin.Flag("include-regex", "Process only log files matched by regex, can't be used with exclude-regex").
		Default("").
		Envar("INCLUDE_REGEX").
		StringVar(&c.includeRegex)

	kingpin.Parse()

	if c.includeRegex != "" && c.excludeRegex != "" {
		log.Fatal("You can not set include and exclude regexs at the same time.")
	}
	if c.excludeRegex != "" {
		c.ExcludeRegex = regexp.MustCompile(c.excludeRegex)
	}
	if c.includeRegex != "" {
		c.IncludeRegex = regexp.MustCompile(c.includeRegex)
	}

	return c
}

// ToString converts config to table formatted multiline string
func (c *Config) ToString() string {
	v := reflect.ValueOf(*c)
	output := "\n"
	for i := 0; i < v.NumField(); i++ {
		if v.Field(i).CanInterface() {
			output += fmt.Sprintf("%s:\t\t'%v'\n", v.Type().Field(i).Name, v.Field(i).Interface())
		}
	}
	return output
}
