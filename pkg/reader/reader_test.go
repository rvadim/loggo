package reader

import (
	"log"
	"os"
	"sync"
	"testing"
	"time"

	"rvadim/loggo/pkg/parser"
	"rvadim/loggo/pkg/storage"
	"rvadim/loggo/pkg/tests"

	"github.com/stretchr/testify/assert"
	"rvadim/loggo/pkg/config"
)

func createTestFile(path string, content string) {
	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		log.Panicln(err)
	}
	defer file.Close()
	_, err = file.WriteString(content)
	if err != nil {
		log.Panicln(err)
	}
}

func renameFile(path string, newpath string) {
	err := os.Rename(path, newpath)
	if err != nil {
		log.Println(err)
	}
}

func deleteFile(path string) {
	err := os.Remove(path)
	if err != nil {
		log.Panicln(err)
	}
}

func TestVeryLongLine(t *testing.T) {
	customLog := `{"log":"{\"a\": 1, \"b\": \"str\", \"c\": null}"}
{"log":"{\"a\": 1, \"b\": \"str\", \"c\": null}"}
{"log":"`
	for i := 0; i < 20000; i++ {
		customLog += "0000000000000000000000000000000000000000000000000000000000000"
	}
	customLog += `"}
{"log": "end"}`

	createTestFile("/tmp/loggo-test1.log", customLog)
	defer deleteFile("/tmp/loggo-test1.log")

	transport := &tests.RedisClientMock{}
	transport.Connect("127.0.0.1", "32770", "my-logs")

	registry, _ := storage.NewRegistryFile("/tmp/test.db", 1)
	defer registry.Close()
	defer deleteFile("/tmp/test.db")
	ch := make(chan bool)
	wg := &sync.WaitGroup{}
	p := parser.New(make(map[string]interface{}))
	r := InitReader("/tmp/loggo-test1.log", transport, registry, ch, wg, p, &config.Config{ReaderMaxChunk: 100})
	r.ReaderTimeout = 1000
	go r.ProcessLogFile()
	close(ch)
	wg.Wait()
	buffer := transport.GetBuffer()
	assert.Equal(t, 26, len(buffer[0]))
	assert.Equal(t, 26, len(buffer[1]))
	assert.Equal(t, 1220010, len(buffer[2]))
	assert.Equal(t, 13, len(buffer[3]))
}

func TestReader(t *testing.T) {
	createTestFile("/tmp/loggo-test1.log", `{"log":"hello world"}
{"log":"{\"a\": 1, \"b\": \"str\", \"c\": null}"}
{"log":"hello world2"}`)
	defer deleteFile("/tmp/loggo-test1.log")

	transport := &tests.RedisClientMock{}
	transport.Connect("127.0.0.1", "32770", "my-logs")

	registry, _ := storage.NewRegistryFile("/tmp/test.db", 1)
	defer registry.Close()
	defer deleteFile("/tmp/test.db")
	ch := make(chan bool)
	wg := &sync.WaitGroup{}
	p := parser.New(make(map[string]interface{}))
	r := InitReader("/tmp/loggo-test1.log", transport, registry, ch, wg, p, &config.Config{ReaderMaxChunk: 10})
	r.ReaderTimeout = 1000
	go r.ProcessLogFile()
	close(ch)
	wg.Wait()
	buffer := transport.GetBuffer()
	assert.Equal(t, "{\"log\":\"hello world\"}", buffer[0])
	assert.Equal(t, "{\"a\":1,\"b\":\"str\",\"c\":null}", buffer[1])
	assert.Equal(t, "{\"log\":\"hello world2\"}", buffer[2])
}

func TestReaderDeleteFile(t *testing.T) {
	createTestFile("/tmp/loggo-test-delete.log", `{"log":"hello world"}
{"log":"{\"a\": 1, \"b\": \"str\", \"c\": null}"}
{"log":"hello world2"}`)

	transport := &tests.RedisClientMock{}
	transport.Connect("127.0.0.1", "32770", "my-logs")

	registry, _ := storage.NewRegistryFile("/tmp/test-delete.db", 1)
	defer registry.Close()
	defer deleteFile("/tmp/test-delete.db")

	ch := make(chan bool)
	wg := &sync.WaitGroup{}
	p := parser.New(make(map[string]interface{}))
	r := InitReader("/tmp/loggo-test-delete.log", transport, registry, ch, wg, p, &config.Config{ReaderMaxChunk: 10})
	r.ReaderTimeout = 1000 * 1000
	go r.ProcessLogFile()
	time.Sleep(10 * time.Millisecond)
	deleteFile("/tmp/loggo-test-delete.log")
	wg.Wait()
	buffer := transport.GetBuffer()
	position, err := registry.Get("/tmp/loggo-test-delete.log")
	assert.NoError(t, err)
	assert.Equal(t, "", position)
	assert.Equal(t, "{\"log\":\"hello world\"}", buffer[0])
	assert.Equal(t, "{\"a\":1,\"b\":\"str\",\"c\":null}", buffer[1])
	assert.Equal(t, "{\"log\":\"hello world2\"}", buffer[2])
}

func TestReaderRenameFile(t *testing.T) {
	createTestFile("/tmp/loggo-test-rename.log", `{"log":"hello world"}
{"log":"{\"a\": 1, \"b\": \"str\", \"c\": null}"}
{"log":"hello world2"}`)
	defer deleteFile("/tmp/loggo-test-rename.log.1")

	transport := &tests.RedisClientMock{}
	transport.Connect("127.0.0.1", "32770", "my-logs")

	registry, _ := storage.NewRegistryFile("/tmp/test-rename.db", 1)
	defer registry.Close()
	defer deleteFile("/tmp/test-rename.db")

	ch := make(chan bool)
	wg := &sync.WaitGroup{}
	p := parser.New(make(map[string]interface{}))
	r := InitReader("/tmp/loggo-test-rename.log", transport, registry, ch, wg, p, &config.Config{ReaderMaxChunk: 10})
	r.ReaderTimeout = 1000 * 1000
	go r.ProcessLogFile()
	time.Sleep(10 * time.Millisecond)
	// TODO write some data before rename
	renameFile("/tmp/loggo-test-rename.log", "/tmp/loggo-test-rename.log.1")
	wg.Wait()
	buffer := transport.GetBuffer()
	position, err := registry.Get("/tmp/loggo-test-rename.log")
	assert.NoError(t, err)
	assert.Equal(t, "", position)
	assert.Equal(t, "{\"log\":\"hello world\"}", buffer[0])
	assert.Equal(t, "{\"a\":1,\"b\":\"str\",\"c\":null}", buffer[1])
	assert.Equal(t, "{\"log\":\"hello world2\"}", buffer[2])
}

func TestTransportFaliure(t *testing.T) {
	createTestFile("/tmp/loggo-test-rename.log", `{"log":"hello world"}
{"log":"{\"a\": 1, \"b\": \"str\", \"c\": null}"}
{"log":"hello world2"}`)
	defer deleteFile("/tmp/loggo-test-rename.log")

	transport := &tests.RedisClientMock{}
	transport.Connect("broken", "32770", "my-logs")

	registry, _ := storage.NewRegistryFile("/tmp/test-rename.db", 1)
	defer registry.Close()
	defer deleteFile("/tmp/test-rename.db")

	ch := make(chan bool)
	wg := &sync.WaitGroup{}
	p := parser.New(make(map[string]interface{}))
	r := InitReader("/tmp/loggo-test-rename.log", transport, registry, ch, wg, p, &config.Config{ReaderMaxChunk: 10})
	r.ReaderTimeout = 1000 * 1000 * 5
	go r.ProcessLogFile()
	time.Sleep(10 * time.Millisecond)
	close(r.ch)
	wg.Wait()
	position, err := registry.Get("/tmp/loggo-test-rename.log")
	assert.NoError(t, err)
	assert.Equal(t, "", position)
}
