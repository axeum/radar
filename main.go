package main

import (
	"bytes"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/coreos/etcd/client"
	"github.com/fsouza/go-dockerclient"
	"golang.org/x/net/context"
)

var (
	dockerEndpoint = os.Getenv("DOCKER_ENDPOINT")
	etcdEndpoint   = os.Getenv("ETCD_ENDPOINT")
	baseKey        = os.Getenv("ETCD_BASEKEY")
	etcdclient     client.KeysAPI
	dockerclient   *docker.Client
)

func containerInfo(containerID string) *docker.Container {
	container, err := dockerclient.InspectContainer(containerID)
	if err != nil {
		log.Printf("Couldn't inspect container: %s", err)
	}
	return container
}

func containerIP(containerID string) string {
	var stdout bytes.Buffer
	var reader = strings.NewReader("")
	execOptions := docker.CreateExecOptions{
		Container:    containerID,
		AttachStdin:  false,
		AttachStdout: true,
		AttachStderr: false,
		Tty:          false,
		Cmd: []string{"sh", "-c",
			"ifconfig eth0 | grep 'inet addr:' | cut -d: -f2 | awk '{ print $1}'"},
	}
	execStartOptions := docker.StartExecOptions{
		OutputStream: &stdout,
		InputStream:  reader,
		Detach:       false,
	}

	execObj, err := dockerclient.CreateExec(execOptions)
	if err != nil {
		log.Printf("Error: %s", err)
	}

	if err := dockerclient.StartExec(execObj.ID, execStartOptions); err != nil {
		log.Printf("Error: %s", err)
	}

	return strings.Trim(stdout.String(), "\n")
}

func dnsKey(fqdn string) string {
	s := strings.Split(fqdn, ".")
	s = append(s, strings.TrimRight(baseKey, "/"))
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
	return strings.Join(s, "/")
}

func fqdn(containerInfo *docker.Container) string {
	hostName := containerInfo.Config.Hostname
	domainName := containerInfo.Config.Domainname
	fqdnPattern, _ := regexp.Compile(`^([a-z0-9]+(-[a-z0-9]+)*\.)+[a-z]{2,}$`)

	fqdn := strings.Trim(strings.ToLower(strings.Join([]string{hostName, domainName}, ".")), ".")
	if fqdnPattern.MatchString(fqdn) {
		return fqdn
	}
	return ""
}

func addDNSRecord(containerID string) {
	if containerInfo := containerInfo(containerID); containerInfo != nil {
		containerIP := containerIP(containerID)
		containerFqdn := fqdn(containerInfo)
		if containerFqdn != "" && containerIP != "" {
			dnsKey := dnsKey(containerFqdn)
			dnsValue := "{\"host\":\"" + containerIP + "\"}"
			log.Printf("Adding DNS record: %#v -> %#v", dnsKey, dnsValue)
			_, err := etcdclient.Set(context.Background(), dnsKey, dnsValue, nil)
			if err != nil {
				log.Printf("Couldn't create key: %s", err)
			}
		}
	}
}

func removeDNSRecord(containerID string) {
	if containerInfo := containerInfo(containerID); containerInfo != nil {
		containerFqdn := fqdn(containerInfo)
		if containerFqdn != "" {
			log.Printf("Removing DNS record: %#v", containerFqdn)
			dnsKey := dnsKey(containerFqdn)
			_, err := etcdclient.Delete(context.Background(), dnsKey, nil)
			if err != nil {
				log.Printf("Couldn't delete key: %s", err)
			}
		}
	}
}

func handleContainerEvent(event *docker.APIEvents) {
	containerName := event.Actor.Attributes["name"]
	containerID := event.Actor.ID

	switch event.Action {
	case "health_status: healthy":
		if containerInfo := containerInfo(containerID); containerInfo != nil {
			containerType := containerInfo.Config.Labels["bigboat.service.type"]
			if containerType == "net" {
				log.Printf("Container '%s' %v", containerName, event.Action)
				addDNSRecord(containerID)
			}
		}
	case "die":
		log.Printf("Container '%s' %v", containerName, event.Action)
		removeDNSRecord(containerID)
	}
}

func processExistingContainers() {
	filters := make(map[string][]string)
	labels := []string{"bigboat.service.type=net"}
	filters["label"] = labels
	containerOpts := docker.ListContainersOptions{
		Filters: filters,
	}
	containers, err := dockerclient.ListContainers(containerOpts)
	if err != nil {
		log.Printf("Failed to get containers: %s", err)
	}

	log.Println("Processing existing containers")
	for _, container := range containers {
		if container.State == "running" {
			c, _ := dockerclient.InspectContainer(container.ID)
			if c.State.Health.Status == "healthy" {
				go addDNSRecord(container.ID)
			}
		}
	}
	log.Println("All existing containers have been processed")
}

func initializeEtcd() {
	log.Printf("Etcd client connected to: %s", etcdEndpoint)
	cfg := client.Config{Endpoints: []string{etcdEndpoint}}
	c, err := client.New(cfg)
	if err != nil {
		log.Fatalf("Setup of etcd client failed: %s", err)
	}
	etcdclient = client.NewKeysAPI(c)
}

func initializeDocker() {
	log.Printf("Docker client connected to: %s", dockerEndpoint)
	d, err := docker.NewClient(dockerEndpoint)
	if err != nil {
		log.Fatalf("Setup of docker client failed: %s", err)
	}
	dockerclient = d
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	//Setup etcdclient
	initializeEtcd()

	//Setup docker client
	initializeDocker()

	//Process existing containers
	processExistingContainers()

	// Add eventlistener source them to events channel
	events := make(chan *docker.APIEvents)
	dockerclient.AddEventListener(events)

	// Process incoming events
	log.Println("Start listening for docker events")
	for {
		select {
		case event := <-events:
			if event.Type == "container" {
				go handleContainerEvent(event)
			}
		}
	}
}
