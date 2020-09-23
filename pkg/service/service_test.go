package service

import (
	"testing"

	"regexp"

	"os"

	"sync"

	"github.com/stretchr/testify/assert"
	"rvadim/loggo/pkg/config"
	"rvadim/loggo/pkg/docker"
	"rvadim/loggo/pkg/k8s"
	"rvadim/loggo/pkg/reader"
	"rvadim/loggo/pkg/storage"
	"rvadim/loggo/pkg/tests"
)

func TestIsNeedToSpawnProcessAllButRouter(t *testing.T) {
	r, err := storage.NewRegistryFile("/tmp/test.db", 1)
	assert.NoError(t, err)
	defer os.Remove("/tmp/test.db")
	s := Service{cfg: &config.Config{
		ExcludeRegex: regexp.MustCompile("^loggo.*|^deis-router.*"),
	}, registry: r}

	c := &docker.Container{}
	c.Config = docker.ConfigSection{Labels: make(map[string]string)}

	c.Config.Labels[k8s.LabelKubernetesContainerName] = "all-other"
	assert.True(t, s.isNeedToSpawnProcess(c, true))
	c.Config.Labels[k8s.LabelKubernetesContainerName] = "loggo"
	assert.False(t, s.isNeedToSpawnProcess(c, true))
	c.Config.Labels[k8s.LabelKubernetesContainerName] = "deis-router"
	assert.False(t, s.isNeedToSpawnProcess(c, true))
}

func TestIsNeedToSpawnProcessRouterOnly(t *testing.T) {
	r, err := storage.NewRegistryFile("/tmp/test.db", 1)
	assert.NoError(t, err)
	defer os.Remove("/tmp/test.db")
	s := Service{cfg: &config.Config{
		IncludeRegex: regexp.MustCompile("^deis-router.*"),
	}, registry: r}

	c := &docker.Container{}
	c.Config = docker.ConfigSection{Labels: make(map[string]string)}

	c.Config.Labels[k8s.LabelKubernetesContainerName] = "all-other"
	assert.False(t, s.isNeedToSpawnProcess(c, true))
	c.Config.Labels[k8s.LabelKubernetesContainerName] = "loggo"
	assert.False(t, s.isNeedToSpawnProcess(c, true))
	c.Config.Labels[k8s.LabelKubernetesContainerName] = "deis-router"
	assert.True(t, s.isNeedToSpawnProcess(c, true))
}

type FinderMock struct {
}

func (f *FinderMock) GetAllContainers() ([]*docker.Container, error) {
	output := []*docker.Container{
		&docker.Container{
			LogPath: "existed-file.log",
		},
	}
	return output, nil
}

func TestRegistryCleanUp(t *testing.T) {
	r, err := storage.NewRegistryFile("/tmp/test.db", 1)
	assert.NoError(t, err)
	defer os.Remove("/tmp/test.db")
	r.Set("not-existed-file.log", "10")
	r.Set("existed-file.log", "11")

	finder := &FinderMock{}
	assert.NoError(t, err)
	s := Service{cfg: &config.Config{
		DirRereadIntervalSec: 1,
		LogsPath:             "../tests/fixtures",
		ExcludeRegex:         regexp.MustCompile("loggo.*|deis-router.*"),
	},
		registry:  r,
		transport: &tests.RedisClientMock{},
		ch:        make(chan bool),
		finder:    finder,
		waitGroup: &sync.WaitGroup{}}

	go s.registryKeyWatcher()
	s.waitGroup.Add(1)
	s.Stop()
	position, err := r.Get("existed-file.log")
	assert.NoError(t, err)
	assert.Equal(t, "11", position)
	position, err = r.Get("not-existed-file.log")
	assert.NoError(t, err)
	assert.Equal(t, "", position)
	r.Close()
}

func TestGetExtends(t *testing.T) {
	labels := make(map[string]string, 3)
	labels[k8s.LabelKubernetesPodName] = "pod name"
	labels[k8s.LabelKubernetesPodNamespace] = "pod namespace"
	labels[k8s.LabelKubernetesContainerName] = "container name"
	c := &docker.Container{
		ID:      "container id",
		LogPath: "log path",
		Config:  docker.ConfigSection{Labels: labels},
	}
	s := &Service{
		cfg: &config.Config{
			NodeHostname:   "node hostname",
			LogstashPrefix: "logstash prefix",
			DataCenter:     "datacenter name",
			Purpose:        "purpose",
			LogType:        "log type",
		},
	}
	p, err := s.getExtendsForLogs(c)
	assert.NoError(t, err)
	assert.Equal(t, "pod name", p[reader.KubernetesPodName])
	assert.Equal(t, "pod namespace", p[reader.KubernetesNamespaceName])
	assert.Equal(t, "pod namespace", p["namespace"])
	assert.Equal(t, "container name", p[reader.KubernetesContainerName])
	assert.Equal(t, "node hostname", p[reader.KubernetesNodeHostname])
	assert.Equal(t, "container id", p["container_id"])
	assert.Equal(t, "datacenter name", p["dc"])
	assert.Equal(t, "purpose", p["purpose"])
	assert.Equal(t, "log type", p["type"])
	assert.Equal(t, "logstash prefix", p["logstash_prefix"])
}
