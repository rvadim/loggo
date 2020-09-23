package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseLine(t *testing.T) {
	p := New(make(map[string]interface{}))
	out, err := p.ParseLine("{log\":\"hello world\"}")
	assert.NotNil(t, err, "ParseLine not failed on invalid input")
	assert.Equal(t, "{log\":\"hello world\"}", out)

	out, err = p.ParseLine("{\"log\":\"hello world\"}")
	assert.NotNil(t, err)
	assert.Equal(t, "{\"log\":\"hello world\"}", out)

	out, err = p.ParseLine(`{
		"log": "{\"hello\":\"world\",\"a\": 1,\"b\": null}"
		}`)
	assert.NoError(t, err)
	assert.Equal(t, "{\"a\":1,\"b\":null,\"hello\":\"world\"}", out)

}

func TestParser(t *testing.T) {
	i := make(map[string]interface{})
	p := New(i)
	i["dc"] = "nsk"
	out, err := p.ParseLine("{\"log\":\"hello world\"}")
	assert.NotNil(t, err)
	assert.Equal(t, "{\"dc\":\"nsk\",\"log\":\"hello world\"}", out)

	out, err = p.ParseLine(`{
		"log": "{\"hello\":\"world\",\"a\": 1,\"b\": null}"
		}`)
	assert.NoError(t, err)
	assert.Equal(t, "{\"a\":1,\"b\":null,\"dc\":\"nsk\",\"hello\":\"world\"}", out)

	out, err = p.ParseLine(`{
		"log": "{\"hello\":\"world\",\"data\":{\"a\":1},\"a\": 1,\"b\": null}"
		}`)
	assert.NoError(t, err)
	assert.Equal(t, "{\"a\":1,\"b\":null,\"data.a\":1,\"dc\":\"nsk\",\"hello\":\"world\"}", out)

	out, err = p.ParseLine(`{
		"log": "{\"hello\":\"world\",\"data\":{\"a\":1,\"b\":\"test\"},\"a\": 1,\"b\": null}"
		}`)
	assert.NoError(t, err)
	assert.Equal(t, "{\"a\":1,\"b\":null,\"data.a\":1,\"data.b\":\"test\",\"dc\":\"nsk\",\"hello\":\"world\"}", out)

	out, err = p.ParseLine(`{
		"log": "{\"hello\":\"world\",\"data\":{\"a\":1,\"b\":{\"a\":1}},\"a\": 1,\"b\": null}"
		}`)
	assert.NoError(t, err)
	assert.Equal(t, "{\"a\":1,\"b\":null,\"data.a\":1,\"dc\":\"nsk\",\"hello\":\"world\"}", out)

	out, err = p.ParseLine(`{
		"log": "{\"hello\":\"world\",\"data\":{\"a\":1,\"b\":null},\"a\": 1,\"b\": null}"
		}`)
	assert.NoError(t, err)
	assert.Equal(t, "{\"a\":1,\"b\":null,\"data.a\":1,\"data.b\":null,\"dc\":\"nsk\",\"hello\":\"world\"}", out)

	assert.Equal(t, "nsk", p.GetProperty("dc"))
	assert.Equal(t, nil, p.GetProperty("unknown-key"))
}

func TestParseUpstreamResponseTime(t *testing.T) {
	p := New(make(map[string]interface{}))
	out, err := p.ParseLine(`{
		"log": "{\"upstream_response_time\":\"-\"}"
		}`)
	assert.NoError(t, err)
	assert.Equal(t, "{\"upstream_response_time\":\"-\"}", out)

	out, err = p.ParseLine(`{
		"log": "{\"upstream_response_time\":\"0\"}"
		}`)
	assert.NoError(t, err)
	assert.Equal(t, "{\"upstream_response_time\":\"0\",\"upstream_response_time_float\":0}", out)

	out, err = p.ParseLine(`{
		"log": "{\"upstream_response_time\":\"0.009\"}"
		}`)
	assert.NoError(t, err)
	assert.Equal(t, "{\"upstream_response_time\":\"0.009\",\"upstream_response_time_float\":0.009}", out)

	out, err = p.ParseLine(`{
		"log": "{\"upstream_response_time\":\"0.009,1.142\"}"
		}`)
	assert.NoError(t, err)
	assert.Equal(t, "{\"upstream_response_time\":\"0.009,1.142\",\"upstream_response_time_float\":1.142}", out)

	out, err = p.ParseLine(`{
		"log": "{\"upstream_response_time\":\"0.009, 1.142, 1.222\"}"
		}`)
	assert.NoError(t, err)
	assert.Equal(t, "{\"upstream_response_time\":\"0.009, 1.142, 1.222\",\"upstream_response_time_float\":1.222}", out)

	out, err = p.ParseLine(`{
		"log": "{\"upstream_response_time\":1.222}"
		}`)
	assert.NoError(t, err)
	assert.Equal(t, "{\"upstream_response_time\":1.222,\"upstream_response_time_float\":1.222}", out)

	out, err = p.ParseLine(`{
		"log": "{\"upstream_response_time\":11}"
		}`)
	assert.NoError(t, err)
	assert.Equal(t, "{\"upstream_response_time\":11,\"upstream_response_time_float\":11}", out)

	out, err = p.ParseLine(`{
		"log": "{\"upstream_response_time\":[1.142, 1.222]}"
		}`)
	assert.NoError(t, err)
	assert.Equal(t, "{\"upstream_response_time\":[1.142,1.222]}", out)

	out, err = p.ParseLine(`{
		"log": "{\"upstream_response_time\":{\"value\": 1.142}}"
		}`)
	assert.NoError(t, err)
	assert.Equal(t, "{\"upstream_response_time.value\":1.142}", out)
}
