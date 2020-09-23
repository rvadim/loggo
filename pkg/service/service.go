package service

import (
	"log"
	"sync"
	"time"

	"rvadim/loggo/pkg/config"
	"rvadim/loggo/pkg/containerd"
	"rvadim/loggo/pkg/docker"
	"rvadim/loggo/pkg/metrics"
	"rvadim/loggo/pkg/parser"
	"rvadim/loggo/pkg/reader"
	"rvadim/loggo/pkg/storage"
	"rvadim/loggo/pkg/transport"
)

type LogsFinder interface {
	GetAllContainers() ([]*docker.Container, error)
}

// Options store options for service creation
type Options struct {
	Registry          *storage.RegistryFile
	LogsPath          string
	DirRereadInterval int
}

// Service store service vars
type Service struct {
	ch        chan bool
	waitGroup *sync.WaitGroup
	cfg       *config.Config
	registry  *storage.RegistryFile
	transport transport.ITransportClient
	finder    LogsFinder
}

// NewService Make a new Service.
func NewService(c *config.Config, r *storage.RegistryFile, t transport.ITransportClient, finder LogsFinder) *Service {
	s := &Service{
		ch:        make(chan bool),
		waitGroup: &sync.WaitGroup{},
		cfg:       c,
		registry:  r,
		transport: t,
		finder:    finder,
	}
	s.waitGroup.Add(1)
	return s
}

// Stop the service by closing the service's channel.  Block until the service
// is really stopped.
func (s *Service) Stop() {
	log.Println("Stopping, wait for all readers done, then close transport channel...")
	close(s.ch)
	s.waitGroup.Wait()
	err := s.transport.Close()
	if err != nil {
		log.Fatalf("Error during closing transport: '%s'", err.Error())
	}
}

// Start starts the listening directory for new logs and spawn readers for each file
func (s *Service) Start() {
	defer s.waitGroup.Done()
	var isFirstIteration = true
	s.waitGroup.Add(1) // For registryKeyWatcher
	go s.registryKeyWatcher()
	go metrics.ServeHTTPRequests(":8080", "/metrics")
	for {
		select {
		case <-s.ch:
			log.Println("Main: Stopping reading directory", s.cfg.LogsPath)
			return
		default:
		}
		containers, err := s.finder.GetAllContainers()
		if err != nil {
			log.Printf("Unable to get containers list due to '%s'", err)
			s.Stop()
			return
		}
		for _, container := range containers {
			if s.isNeedToSpawnProcess(container, isFirstIteration) {
				extends, err := s.getExtendsForLogs(container)
				if err != nil {
					log.Printf("Error: unable to get extends for path %s, %s", container.LogPath, err)
					continue
				}
				var p IParser
				if container.CRIType == docker.CRI_TYPE_DOCKER {
					p = parser.New(extends)
				} else if container.CRIType == docker.CRI_TYPE_CONTAINERD {
					p = containerd.New(extends)
				}
				log.Printf("Try to init reader for %s, cri-type: %d", container.LogPath, container.CRIType)
				r := reader.InitReader(container.LogPath, s.transport, s.registry, s.ch, s.waitGroup, p, s.cfg)
				if r == nil {
					log.Printf("Init fail for %s", container.LogPath)
					continue
				}
				go r.ProcessLogFile()
			}
		}
		time.Sleep(time.Duration(s.cfg.DirRereadIntervalSec) * time.Second)
		isFirstIteration = false
	}
}

func (s *Service) registryKeyWatcher() {
	log.Println("Starting registry key watcher")
	defer s.waitGroup.Done()
	for {
		containers, _ := s.finder.GetAllContainers()
		keys, err := s.registry.GetAllKeys()
		if err != nil {
			log.Println("Unable to read registry", err)
		} else {
			for _, key := range keys {
				if !stringInContainerSlice(key, containers) {
					log.Printf("Delete from registry: %s", key)
					s.registry.Delete(key)
				}
			}
		}
		select {
		case <-s.ch:
			log.Println("Stop registry key watcher")
			return
		default:
		}
		time.Sleep(time.Duration(s.cfg.DirRereadIntervalSec) * time.Second)
	}
}

func stringInContainerSlice(item string, slice []*docker.Container) bool {
	for _, s := range slice {
		if s.LogPath == item {
			return true
		}
	}
	return false
}

func (s *Service) getExtendsForLogs(c *docker.Container) (parser.Properties, error) {
	out := make(map[string]interface{})
	out[reader.KubernetesPodName] = c.GetPodName()
	out[reader.KubernetesNamespaceName] = c.GetPodNamespace()
	out["namespace"] = c.GetPodNamespace()
	out[reader.KubernetesContainerName] = c.GetName()
	out[reader.KubernetesNodeHostname] = s.cfg.NodeHostname
	out["container_id"] = c.ID
	out["dc"] = s.cfg.DataCenter
	out["purpose"] = s.cfg.Purpose
	out["type"] = s.cfg.LogType
	out["logstash_prefix"] = s.cfg.LogstashPrefix

	return out, nil
}

func (s *Service) isNeedToSpawnProcess(c *docker.Container, isFirstIteration bool) bool {
	if s.cfg.IncludeRegex != nil && !s.isIncluded(c.GetName()) {
		return false
	}
	if s.cfg.ExcludeRegex != nil && s.isExcluded(c.GetName()) {
		return false
	}
	position, err := s.registry.Get(c.LogPath)
	if err != nil {
		position = ""
	}
	if position == "" || isFirstIteration {
		return true
	}
	return false
}

func (s *Service) isExcluded(name string) bool {
	return s.cfg.ExcludeRegex.MatchString(name)
}

func (s *Service) isIncluded(name string) bool {
	return s.cfg.IncludeRegex.MatchString(name)
}
