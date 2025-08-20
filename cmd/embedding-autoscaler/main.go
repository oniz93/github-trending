package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"
	"github.com/joho/godotenv"
)

// --- Configuration ---
var (
	composeProjectName   = getEnv("COMPOSE_PROJECT_NAME", "github-trending")
	serviceToScaleName   = "embedding-api-instance"
	serviceToScale       = fmt.Sprintf("%s_%s", composeProjectName, serviceToScaleName)
	targetPort           = 80
	maxReplicas, _       = strconv.Atoi(getEnv("EMBEDDING_API_MAX_INSTANCES", "3"))
	idleTimeoutSeconds, _ = strconv.Atoi(getEnv("EMBEDDING_API_IDLE_TIMEOUT", "600"))
)

// --- Globals ---
var (
	dockerClient        *client.Client
	lastRequestTime     time.Time
	mutex               sync.Mutex
	instanceIPs         []string
	nextInstanceIndex   int
	concurrentRequests  int
)

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func main() {
	godotenv.Load()

	var err error
	dockerClient, err = client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatalf("Failed to create Docker client: %v", err)
	}

	go scaleDownOnIdle()

	server := &http.Server{
		Addr:    ":80",
		Handler: http.HandlerFunc(proxyHandler),
	}

	log.Println("Autoscaler server started on :80")
	log.Fatal(server.ListenAndServe())
}

func proxyHandler(w http.ResponseWriter, r *http.Request) {
	mutex.Lock()
	lastRequestTime = time.Now()
	concurrentRequests++

	service, err := getService(serviceToScale)
	if err != nil {
		log.Printf("Error getting service: %v", err)
		http.Error(w, "Service not found", http.StatusInternalServerError)
		mutex.Unlock()
		return
	}

	replicas := getServiceReplicas(service)

	if uint64(concurrentRequests) > replicas && replicas < uint64(maxReplicas) {
		newReplicas := replicas + 1
		log.Printf("High load detected. Scaling up to %d instances.", newReplicas)
		scaleService(service, newReplicas)
	} else if replicas == 0 {
		log.Println("Scaling up to 1 instance.")
		scaleService(service, 1)
	}

	instanceIPs, err = getServiceInstanceIPs(service)
	if err != nil || len(instanceIPs) == 0 {
		log.Printf("Error getting instance IPs or no instances found: %v", err)
		http.Error(w, "No running instances found", http.StatusServiceUnavailable)
		mutex.Unlock()
		return
	}

	targetIP := instanceIPs[nextInstanceIndex%len(instanceIPs)]
	nextInstanceIndex++
	mutex.Unlock()

	targetUrl, _ := url.Parse(fmt.Sprintf("http://%s:%d", targetIP, targetPort))
	proxy := httputil.NewSingleHostReverseProxy(targetUrl)

	proxy.ServeHTTP(w, r)

	mutex.Lock()
	concurrentRequests--
	mutex.Unlock()
}

func scaleDownOnIdle() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		mutex.Lock()
		if !lastRequestTime.IsZero() && time.Since(lastRequestTime) > time.Duration(idleTimeoutSeconds)*time.Second {
			service, err := getService(serviceToScale)
			if err != nil {
				log.Printf("Error getting service for scale down: %v", err)
			} else if getServiceReplicas(service) > 0 {
				log.Println("Scaling down to 0 due to inactivity.")
				scaleService(service, 0)
				lastRequestTime = time.Time{}
			}
		}
		mutex.Unlock()
	}
}

func getService(serviceName string) (*swarm.Service, error) {
	service, _, err := dockerClient.ServiceInspectWithRaw(context.Background(), serviceName, types.ServiceInspectOptions{})
	if err != nil {
		return nil, err
	}
	return &service, nil
}

func getServiceReplicas(service *swarm.Service) uint64 {
	if service.Spec.Mode.Replicated != nil {
		return *service.Spec.Mode.Replicated.Replicas
	}
	return 0
}

func scaleService(service *swarm.Service, replicas uint64) {
	serviceSpec := service.Spec
	serviceSpec.Mode.Replicated.Replicas = &replicas

	_, err := dockerClient.ServiceUpdate(context.Background(), service.ID, service.Version, serviceSpec, types.ServiceUpdateOptions{})
	if err != nil {
		log.Printf("Error scaling service: %v", err)
	}

	// Wait for replicas to be ready
	for {
		listFilters := filters.NewArgs()
		listFilters.Add("service", service.ID)
		listFilters.Add("desired-state", "running")
		tasks, err := dockerClient.TaskList(context.Background(), types.TaskListOptions{Filters: listFilters})
		if err == nil && len(tasks) == int(replicas) {
			break
		}
		time.Sleep(1 * time.Second)
	}
}

func getServiceInstanceIPs(service *swarm.Service) ([]string, error) {
	listFilters := filters.NewArgs()
	listFilters.Add("service", service.ID)
	listFilters.Add("desired-state", "running")
	tasks, err := dockerClient.TaskList(context.Background(), types.TaskListOptions{Filters: listFilters})
	if err != nil {
		return nil, err
	}

	var ips []string
	for _, task := range tasks {
		for _, network := range task.NetworksAttachments {
			ips = append(ips, strings.Split(network.Addresses[0], "/")[0])
		}
	}
	return ips, nil
}
