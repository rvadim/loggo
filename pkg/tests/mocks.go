package tests

import "fmt"

// RedisClientMock mock for redis client
type RedisClientMock struct {
	buffer []string
	closed bool
	broken bool
}

// Connect do nothing in mock
func (r *RedisClientMock) Connect(hostname string, port string, key string) {
	r.broken = false
	if hostname == "broken" {
		r.broken = true
	}
	r.closed = false
}

// DeliverMessages store data in buffer
func (r *RedisClientMock) DeliverMessages(data []string) error {
	if r.broken {
		return fmt.Errorf("Exception (504) Reason: \"channel/connection is not open\"")
	}
	r.buffer = data
	return nil
}

// Close do nothing in mock
func (r *RedisClientMock) Close() error {
	if r.broken {
		return fmt.Errorf("Exception (504) Reason: \"channel/connection is not open\"")
	}
	r.closed = true
	return nil
}

// GetBuffer return content of buffer in mock
func (r *RedisClientMock) GetBuffer() []string {
	return r.buffer
}

// GetClosed return state of mock connection
func (r *RedisClientMock) GetClosed() bool {
	return r.closed
}
