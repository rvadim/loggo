package redisclient

import (
	"context"
	"encoding/json"
	"github.com/go-redis/redis/v8"
	"log"
)

// RedisClient for redis transport
type RedisClient struct {
	client *redis.Client
	key    string
}

// New connect to redis
func New(hostname string, key string, password string) (*RedisClient, error) {
	return &RedisClient{
		key: key,
		client: redis.NewClient(&redis.Options{
			Addr:     hostname,
			Password: password,
			DB:       0, // use default DB
		}),
	}, nil
}

// DeliverMessages send array of strings to redis
func (r *RedisClient) DeliverMessages(data []string) error {
	var newData []interface{}
	for _, value := range data {
		validate(value)
		newData = append(newData, value)
	}
	return r.client.RPush(context.Background(), r.key, newData...).Err()
}

// ReceiveMessage returns message from list
func (r *RedisClient) ReceiveMessage() ([]byte, error) {
	msg := r.client.LPop(context.Background(), r.key)
	if msg.Err() != nil {
		return nil, msg.Err()
	}

	return msg.Bytes()
}

// Close close connection
func (r *RedisClient) Close() error {
	return r.client.Close()
}

type Message struct {
	DC string `json:"dc"`
}

func validate(data string) {
	err := json.Unmarshal([]byte(data), &Message{})
	if err != nil {
		log.Printf("Unable to validate message: %s", data)
	}
}