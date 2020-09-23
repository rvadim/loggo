package firehose

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws/session"
	fh "github.com/aws/aws-sdk-go/service/firehose"
)

type FireHose struct {
	client         *fh.Firehose
	deliveryStream string
}

func (f *FireHose) DeliverMessages(strings []string) error {
	rb := &fh.PutRecordBatchInput{}
	rb.DeliveryStreamName = &f.deliveryStream
	var records []*fh.Record
	for _, s := range strings {
		records = append(records, &fh.Record{Data: []byte(s)})
	}
	rb.SetRecords(records)
	_, err := f.client.PutRecordBatch(rb)
	if err != nil {
		return fmt.Errorf("unable to delive message to delivery stream %s, %w", f.deliveryStream, err)
	}
	return nil
}

func (f *FireHose) Close() error {
	return nil
}

func New(deliveryStream string) (*FireHose, error) {
	s, err := session.NewSession()
	if err != nil {
		return &FireHose{}, fmt.Errorf("unable to create new aws session, %w", err)
	}
	return &FireHose{
		deliveryStream: deliveryStream,
		client:         fh.New(s),
	}, nil
}
