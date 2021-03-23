package docker

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"rvadim/loggo/pkg/k8s"
)

// * Read all directoreis in /var/log/pods/
// * In each directory find all symlinks
// * For each symlink get destination file path
// * Read container id from destination file path
// * Get:
//    * namespace                   Config.Labels."io.kubernetes.pod.namespace"
//    * docker.container_id         ID
//    * kubernetes.pod_name         Config.Labels."io.kubernetes.pod.name"
//    * kubernetes.namespace_name   Config.Labels."io.kubernetes.pod.namespace"
//    * kubernetes.container_name   Config.Labels."io.kubernetes.container.name"

const (
	configFileName      = "config.v2.json"
	CRI_TYPE_CONTAINERD = 0
	CRI_TYPE_DOCKER     = 1
)

// Container store container configuration parameters
type Container struct {
	ID      string        `json:"ID"`
	LogPath string        `json:"LogPath"`
	Config  ConfigSection `json:"Config"`
	CRIType int
}

// ConfigSection store Config from container configuration parameters
type ConfigSection struct {
	Labels map[string]string
}

// Finder seek for logs in requested logPath and resolve links
type Finder struct {
	logsPath string
	mu       sync.Mutex
}

// GetPodName returns container pod name or empty string
func (c *Container) GetPodName() string {
	return c.getLabelValue(k8s.LabelKubernetesPodName)
}

// GetPodNamespace returns container pod namespace or empty string
func (c *Container) GetPodNamespace() string {
	return c.getLabelValue(k8s.LabelKubernetesPodNamespace)
}

// GetName returns container name or empty string
func (c *Container) GetName() string {
	return c.getLabelValue(k8s.LabelKubernetesContainerName)
}

func (c *Container) getLabelValue(label string) string {
	if value, ok := c.Config.Labels[label]; ok {
		return value
	}
	return ""
}

// NewFinder constructs new Finder by path
func NewFinder(path string) (*Finder, error) {
	log.Printf("Finder work in path '%s'", path)
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	log.Printf("Absolute path for finder '%s'", absPath)
	return &Finder{
		logsPath: absPath,
	}, nil
}

// GetAllContainers seek and return all Containers
func (f *Finder) GetAllContainers() ([]*Container, error) {
	var containers []*Container

	dirs, err := getAllDirectories(f.logsPath)
	if err != nil {
		log.Printf("Unable to get all directories in '%s', due to '%s'", f.logsPath, err)
		return containers, err
	}
	for _, subdir := range dirs {
		subdirs, err := getAllDirectories(subdir)
		if err != nil {
			log.Printf("Unable to get all directories in '%s', due to '%s'", f.logsPath, err)
			continue
		}

		for _, dir := range subdirs {
			files, err := getAllFiles(dir)
			if err != nil {
				log.Printf("Error: unable to read dir: %s", dir)
				continue
			}

			for _, file := range files {
				var container *Container
				symlink, err := isSymlink(file)
				if err != nil {
					log.Printf("Error: Unable to detect file symlink or not %s, %s", file, err)
					continue
				}
				if symlink {
					container, err = f.buildDockerContainerLog(file)
					if err != nil {
						log.Print(err)
						continue
					}
				} else {
					container, err = f.buildContainerDContainerLog(file)
					if err != nil {
						log.Print(err)
						continue
					}
					container.CRIType = CRI_TYPE_CONTAINERD
				}
				containers = append(containers, container)
			}
		}
	}
	return containers, nil
}

func (f *Finder) buildContainerDContainerLog(file string) (*Container, error) {
	path := filepath.Dir(file)
	containerName := filepath.Base(path)
	podString := filepath.Base(filepath.Dir(path))

	output := strings.Split(podString, "_")
	namespace := output[0]
	pod := output[1]
	id := output[2]

	return &Container{
		CRIType: CRI_TYPE_CONTAINERD,
		ID:      id,
		LogPath: file,
		Config: ConfigSection{
			Labels: map[string]string{
				k8s.LabelKubernetesPodName:       pod,
				k8s.LabelKubernetesPodNamespace:  namespace,
				k8s.LabelKubernetesContainerName: containerName,
			},
		},
	}, nil
}

func (f *Finder) buildDockerContainerLog(link string) (*Container, error) {
	path, err := f.resolveSymlink(link)
	if err != nil {
		return nil, fmt.Errorf("unable to read link: %s, %w", link, err)
	}
	configPath, err := getConfigFilePath(path)
	if err != nil {
		return nil, fmt.Errorf("get config for logfile: %s, %w", path, err)
	}
	container, err := unserializeConfigFile(configPath)
	container.LogPath = path
	if err != nil {
		return nil, err
	}

	container.CRIType = CRI_TYPE_DOCKER
	return container, nil
}

func getAllDirectories(path string) ([]string, error) {
	var output []string
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return output, err
	}
	for _, file := range files {
		if file.IsDir() {
			output = append(output, filepath.Join(path, file.Name()))
		}
	}
	return output, nil
}

func isSymlink(path string) (bool, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return false, err
	}
	return info.Mode()&os.ModeSymlink == os.ModeSymlink, nil
}

func getAllFiles(path string) ([]string, error) {
	var output []string
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return output, err
	}
	for _, file := range files {
		//if file.Mode()&os.ModeSymlink != 0 {
		output = append(output, filepath.Join(path, file.Name()))
		//}
	}
	return output, nil
}

func getConfigFilePath(logfile string) (string, error) {
	path, err := filepath.Abs(filepath.Dir(logfile))
	if err != nil {
		return "", err
	}
	return filepath.Join(path, configFileName), nil
}

func (f *Finder) resolveSymlink(path string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	oldPath, err := os.Getwd()
	if err != nil {
		return "", err
	}
	newPath, err := filepath.Abs(filepath.Dir(path))
	if err != nil {
		return "", err
	}
	target, err := os.Readlink(path)
	if err != nil {
		return "", err
	}
	err = os.Chdir(newPath)
	if err != nil {
		return "", err
	}
	defer os.Chdir(oldPath)
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return "", err
	}
	return absTarget, nil
}

// UnserializeConfigFile read file by configPath and return ContainerConfigFile struct
func unserializeConfigFile(configPath string) (*Container, error) {
	containerConfigFile := &Container{}
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return containerConfigFile, err
	}
	err = json.Unmarshal(data, containerConfigFile)
	return containerConfigFile, err
}
