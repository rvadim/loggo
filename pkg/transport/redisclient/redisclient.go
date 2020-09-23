package redisclient

import (
	"log"
	"time"

	"github.com/garyburd/redigo/redis"
	"github.com/pkg/errors"
)

// RedisClient for redis transport
type RedisClient struct {
	conn     redis.Conn
	key      string
	hostname string
	tries    int
	timeout  int
}

// New connect to redis
func New(hostname string, key string) (*RedisClient, error) {
	r := &RedisClient{
		key:      key,
		hostname: hostname,
		tries:    10,
		timeout:  1,
	}
	err := r.connect()
	if err != nil {
		return r, err
	}

	return r, nil
}

func (r *RedisClient) connect() error {
	var err error

	for i := 0; i < r.tries; i++ {
		r.conn, err = redis.Dial("tcp", r.hostname)
		if err != nil {
			log.Printf("Try #%d, Unable to init redis client. %s, retry after timeout %d", i, err, r.timeout)
			time.Sleep(time.Duration(r.timeout) * time.Second)
			continue
		}

		log.Println("Connection to redis initialized.")
		return nil
	}

	return errors.Wrap(err, "Unable to connect to redis")
}

// DeliverMessages send array of strings to redis
func (r *RedisClient) DeliverMessages(data []string) error {
	var newData []interface{}
	var err error
	newData = append(newData, r.key)
	for _, value := range data {
		newData = append(newData, value)
	}
	if r.conn != nil {
		_, err = r.conn.Do("RPUSH", newData...)
	}
	// INFO: v.reyder: Если при первой отправке мы получили ошибку, то просто пытаемся переконнектится и отправить данные ещё раз,
	// в случае когда, соединения небыло(повторый заход в DeliverMessages без соединения), тоже пытаемся переподключится.
	if err != nil || r.conn == nil {
		errConn := r.connect()
		if errConn != nil {
			// INFO: v.reyder: Если не получилось подулючится возвращаем наверх ошибку подключения.
			return errConn
		}
		// INFO: v.reyder: та самая отправка ещё раз
		return r.DeliverMessages(data)
	}
	return nil
}

// ReceiveMessage returns message from list
func (r *RedisClient) ReceiveMessage() ([]byte, error) {
	message, err := redis.Bytes(r.conn.Do("LPOP", r.key))
	if err != nil {
		return nil, err
	}

	return message, nil
}

// Close close connection
func (r *RedisClient) Close() error {
	if r.conn != nil {
		return r.conn.Close()
	}
	return nil
}
