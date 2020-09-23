package docker

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

type podsEnv struct {
	rootDir          string
	allContainersDir string
	containerDir     string
	fileName         string
}

type dockerEnv struct {
	rootDir      string
	containerDir string
	fileName     string
}

type TestContainer struct {
	namespace     string
	podName       string
	containerID   string
	containerName string
}

type testEnvironment struct {
	container TestContainer
	podsEnv   podsEnv
	dockerEnv dockerEnv
	tempDir   string
}

func setUp(t *testing.T, envType string) testEnvironment {
	c := TestContainer{
		namespace:     "logging",
		podName:       "loggo-staging-5875c7c554-4zxh6",
		containerID:   "17860730-09a4-4296-9670-121a21c70799",
		containerName: "loggo",
	}

	te := testEnvironment{
		container: c,
		tempDir:   filepath.Join(os.TempDir(), "loggo-tests"),
		podsEnv:   podsEnv{},
		dockerEnv: dockerEnv{},
	}

	// Define catalogs and files
	te.podsEnv.rootDir = filepath.Join(te.tempDir, "pods")
	te.podsEnv.allContainersDir = filepath.Join(te.podsEnv.rootDir,
		fmt.Sprintf("%s_%s_%s", c.namespace, c.podName, c.containerID))
	te.podsEnv.containerDir = filepath.Join(te.podsEnv.allContainersDir, c.containerName)
	te.podsEnv.fileName = "0.log"

	if envType == "docker" {
		te.dockerEnv.rootDir = filepath.Join(te.tempDir, "containers")
		te.dockerEnv.containerDir = filepath.Join(te.dockerEnv.rootDir, c.containerID)
		te.dockerEnv.fileName = fmt.Sprintf("%s-json.log", c.containerID)
	}

	// Create catalogs
	err := os.MkdirAll(te.podsEnv.containerDir, 0755)
	assert.NoError(t, err)
	if envType == "docker" {
		err = os.MkdirAll(te.dockerEnv.containerDir, 0755)
		assert.NoError(t, err)
	}

	// Create files
	if envType == "docker" {
		logContent := []byte(`{"log":"Hello world"}`)
		configContent := []byte(fmt.Sprintf(`{
    "ID":"%s",
    "Config": {
      "Labels":{
        "io.kubernetes.pod.namespace":"%s",
        "io.kubernetes.pod.name":"%s",
        "io.kubernetes.container.name": "%s"
      }
    }
  }`, te.container.containerID, te.container.namespace, te.container.podName, te.container.containerName))

		ioutil.WriteFile(filepath.Join(te.dockerEnv.containerDir, te.dockerEnv.fileName), logContent, 0700)
		ioutil.WriteFile(filepath.Join(te.dockerEnv.containerDir, configFileName), configContent, 0700)
		os.Symlink(filepath.Join(te.dockerEnv.containerDir, te.dockerEnv.fileName),
			filepath.Join(te.podsEnv.containerDir, te.podsEnv.fileName))
	} else {
		logContent := []byte(`2020-09-13T10:49:23.232671677Z stdout F {"Hello": "World!"}`)
		ioutil.WriteFile(filepath.Join(te.podsEnv.containerDir, te.podsEnv.fileName), logContent, 0700)
	}

	return te
}

func tearDown(te testEnvironment) {
	err := os.RemoveAll(te.tempDir)
	if err != nil {
		log.Fatal(err)
	}
}

func TestFunctions(t *testing.T) {
	te := setUp(t, "docker")
	defer tearDown(te)

	dirs, err := getAllDirectories(te.podsEnv.rootDir)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(dirs))
	assert.Equal(t, te.podsEnv.allContainersDir, dirs[0])

	dirs, err = getAllDirectories(te.podsEnv.allContainersDir)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(dirs))
	assert.Equal(t, te.podsEnv.containerDir, dirs[0])

	files, err := getAllFiles(te.podsEnv.containerDir)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(files))
	assert.Equal(t, filepath.Join(te.podsEnv.containerDir, te.podsEnv.fileName), files[0])

	actualPath, err := os.Readlink(files[0])
	assert.NoError(t, err)
	assert.Equal(t, filepath.Join(te.dockerEnv.containerDir, te.dockerEnv.fileName), actualPath)

	configPath, err := getConfigFilePath(actualPath)
	assert.NoError(t, err)

	cfg, err := unserializeConfigFile(configPath)
	assert.NoError(t, err)
	assert.Equal(t, te.container.containerID, cfg.ID)
	assert.Equal(t, te.container.namespace, cfg.GetPodNamespace())
	assert.Equal(t, te.container.podName, cfg.GetPodName())
	assert.Equal(t, te.container.containerName, cfg.GetName())
}

func TestLogsFinderDocker(t *testing.T) {
	te := setUp(t, "docker")
	defer tearDown(te)

	f, err := NewFinder(te.podsEnv.rootDir)
	assert.NoError(t, err)

	containers, err := f.GetAllContainers()
	assert.NoError(t, err)
	assert.Equal(t, 1, len(containers))
	c := containers[0]
	assert.Equal(t, te.container.containerName, c.GetName())
	assert.Equal(t, te.container.podName, c.GetPodName())
	assert.Equal(t, te.container.namespace, c.GetPodNamespace())
	assert.Equal(t, te.container.containerID, c.ID)
	assert.Equal(t, CRI_TYPE_DOCKER, c.CRIType)
}

func TestLogsFinderContainerD(t *testing.T) {
	te := setUp(t, "containerd")
	defer tearDown(te)

	f, err := NewFinder(te.podsEnv.rootDir)
	assert.NoError(t, err)

	containers, err := f.GetAllContainers()
	assert.NoError(t, err)
	assert.Equal(t, 1, len(containers))
	c := containers[0]
	assert.Equal(t, te.container.containerName, c.GetName())
	assert.Equal(t, te.container.podName, c.GetPodName())
	assert.Equal(t, te.container.namespace, c.GetPodNamespace())
	assert.Equal(t, te.container.containerID, c.ID)
	assert.Equal(t, CRI_TYPE_CONTAINERD, c.CRIType)
}
