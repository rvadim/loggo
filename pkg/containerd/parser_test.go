package containerd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseLine(t *testing.T) {
	p := New(make(map[string]interface{}))
	out, err := p.ParseLine("{log\":\"hello world\"}")
	assert.NotNil(t, err, "ParseLine not failed on invalid input")
	assert.Equal(t, "{log\":\"hello world\"}", out)

	out, err = p.ParseLine("2020-09-10T07:00:03.585507743Z stdout F my message")
	assert.NotNil(t, err)
	assert.Equal(t, "{\"log\":\"my message\",\"stream\":\"stdout\",\"time\":\"2020-09-10T07:00:03.585507743Z\"}", out)

	out, err = p.ParseLine(`2020-09-10T07:00:03.585507743Z stdout F {"hello":"world","a": 1,"b": null}`)
	assert.NoError(t, err)
	assert.Equal(t, `{"a":1,"b":null,"hello":"world","stream":"stdout","time":"2020-09-10T07:00:03.585507743Z"}`, out)

	out, err = p.ParseLine(`2020-09-19T11:57:36.498638614Z stderr F I0919 11:57:36.498396       1 binarylog.go:274] rpc: flushed binary log to ""

`)
	assert.NotNil(t, err)
	assert.Equal(t, `{"log":"I0919 11:57:36.498396       1 binarylog.go:274] rpc: flushed binary log to \"\"\n\n","stream":"stderr","time":"2020-09-19T11:57:36.498638614Z"}`, out)

}
