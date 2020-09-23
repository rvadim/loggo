package transport

// ITransportClient common interface for transport
type ITransportClient interface {
	DeliverMessages([]string) error
	Close() error
}
