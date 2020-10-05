package reader

import (
	"bufio"
	"io"
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"

	"rvadim/loggo/pkg/config"
	"rvadim/loggo/pkg/metrics"
	"rvadim/loggo/pkg/storage"
	"rvadim/loggo/pkg/transport"
)

type IParser interface {
	GetProperty(key string) interface{}
	ParseLine(line string) (string, error)
}

// KubernetesPodName name of field
const KubernetesPodName = "kubernetes.pod_name"

// KubernetesNamespaceName name of field
const KubernetesNamespaceName = "kubernetes.namespace_name"

// KubernetesContainerName name of field
const KubernetesContainerName = "kubernetes.container_name"

// KubernetesNodeHostname name of field
const KubernetesNodeHostname = "kubernetes.node_hostname"

// Reader common struct for log reader
type Reader struct {
	registry      *storage.RegistryFile
	filePath      string
	t             transport.ITransportClient
	file          *os.File
	pos           int64
	maxChunk      int
	ReaderTimeout time.Duration
	ch            chan bool
	waitGroup     *sync.WaitGroup
	watcher       *fsnotify.Watcher
	parser        IParser
}

// InitReader initialize registry and transport
func InitReader(path string, t transport.ITransportClient,
	registry *storage.RegistryFile, ch chan bool, wg *sync.WaitGroup, p IParser, c *config.Config) *Reader {
	r := &Reader{
		ch:            ch,
		waitGroup:     wg,
		registry:      registry,
		filePath:      path,
		maxChunk:      c.ReaderMaxChunk,
		ReaderTimeout: time.Duration(c.ReaderTimeoutSec) * time.Second,
		parser:        p,
		t:             t,
	}
	r.pos = r.getPosition()
	var err error
	r.file, err = os.OpenFile(r.filePath, os.O_RDONLY, 0600)
	if err != nil {
		log.Printf("Unable to open file '%s' for reading, %s", r.filePath, err)
		r.registry.Delete(r.filePath)
		return nil
	}
	r.watcher, err = fsnotify.NewWatcher()
	if err != nil {
		log.Printf("Unable init watching for file '%s', %s", r.filePath, err)
		r.registry.Delete(r.filePath)
		return nil
	}
	r.watcher.Add(r.filePath)
	wg.Add(1)
	return r
}

func (r *Reader) getPosition() int64 {
	strPos, err := r.registry.Get(r.filePath)
	if err != nil || strPos == "" {
		return 0
	}
	pos, err := strconv.ParseInt(strPos, 10, 64)
	if err != nil {
		return 0
	}
	return pos
}

func (r *Reader) setPosition(pos int64) {
	err := r.registry.Set(r.filePath, strconv.FormatInt(pos, 10))
	if err != nil {
		log.Fatalf("Unable to r.Set(%s, %d)", r.filePath, pos)
	}
}

// ProcessLogFile process log file
func (r *Reader) ProcessLogFile() {
	defer r.waitGroup.Done()
	defer r.file.Close()
	defer r.watcher.Close()
	lastIteration := false
	namespace := ""
	podName := ""
	containerName := ""
	if val, ok := r.parser.GetProperty(KubernetesPodName).(string); ok {
		podName = val
	}
	if val, ok := r.parser.GetProperty(KubernetesContainerName).(string); ok {
		containerName = val
	}
	if val, ok := r.parser.GetProperty(KubernetesNamespaceName).(string); ok {
		namespace = val
	}
	for {
		// Workaround for https://github.com/fsnotify/fsnotify/issues/194
		_, err := os.Stat(r.filePath)
		if err != nil && !lastIteration {
			log.Printf("File not present on fs, remove '%s' from registry", r.filePath)
			r.registry.Delete(r.filePath)
			return
		}
		pos, data, err := r.ReadDataReadBytes(r.file, r.pos, r.maxChunk)
		if err != nil {
			log.Printf("Important: Unable to read file %s, from position %d, %s", r.file.Name(), r.pos, err)
			continue
		}
		if len(data) != 0 {
			err = r.t.DeliverMessages(data)
			if err != nil {
				log.Printf("%s: Unable to send data %s, sleep for %s and start processing again", r.filePath, err, r.ReaderTimeout)
				time.Sleep(r.ReaderTimeout)
				select {
				case <-r.ch:
					log.Printf("Stop reading '%s' due to channel closed", r.filePath)
					return
				default:
				}
				continue
			}
		}
		metrics.LogMessageCount.WithLabelValues(namespace, podName, containerName).Add(float64(len(data)))
		r.setPosition(pos)
		r.pos = pos
		if len(data) < r.maxChunk {
			if lastIteration {
				r.registry.Delete(r.filePath)
				return
			}
			time.Sleep(r.ReaderTimeout)
		}

		select {
		case <-r.ch:
			log.Println("Stop reading", r.filePath)
			return
		case event := <-r.watcher.Events:
			if event.Op == fsnotify.Rename {
				lastIteration = true
			}
			if event.Op == fsnotify.Remove {
				// FIXME Never executed due to https://github.com/fsnotify/fsnotify/issues/194
				r.registry.Delete(r.filePath)
				return
			}
		case err := <-r.watcher.Errors:
			log.Printf("Watcher error '%s': %s", r.filePath, err)
		default:
		}
	}
}

// ReadDataReadBytes read from start position and return max lines from file
func (r *Reader) ReadDataReadBytes(input io.ReadSeeker, start int64, max int) (int64, []string, error) {
	if _, err := input.Seek(start, 0); err != nil {
		return 0, nil, err
	}
	var buffer []string
	reader := bufio.NewReader(input)
	pos := start
	for i := 0; i <= max; i++ {
		data, err := reader.ReadBytes('\n')
		if len(data) == 0 && err == io.EOF {
			break
		}
		pos += int64(len(data))
		if err == nil || err == io.EOF {
			out, _ := r.parser.ParseLine(string(data))
			buffer = append(buffer, out)
		} else if err != nil {
			if err != io.EOF {
				return 0, nil, err
			}
			break
		}
	}
	return pos, buffer, nil
}
