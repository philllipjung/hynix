package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

const (
	yunikornAPI = "http://localhost:9080"
	uiPort      = 8082
)

func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}

func proxyHandler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	// Root - redirect to Yunikorn UI
	if path == "/" {
		http.Redirect(w, r, "/yunikorn-ui.html", http.StatusFound)
		return
	}

	// Static files - serve from docs directory
	if path == "/yunikorn-ui.html" || path == "/index.html" {
		http.ServeFile(w, r, "/root/hynix/docs/yunikorn-ui.html")
		return
	}

	// Spark Metrics UI
	if path == "/spark-metrics-ui.html" {
		http.ServeFile(w, r, "/root/hynix/docs/spark-metrics-ui.html")
		return
	}

	// OpenSearch Discovery UI
	if path == "/opensearch-discovery-ui.html" {
		http.ServeFile(w, r, "/root/hynix/docs/opensearch-discovery-ui.html")
		return
	}

	// Kubernetes API proxy (for metrics dashboard) - use kubectl
	if strings.HasPrefix(path, "/api/api/v1/") {
		k8sProxyHandler(w, r, path)
		return
	}

	// OpenSearch API proxy
	if strings.HasPrefix(path, "/api/opensearch") {
		opensearchProxyHandler(w, r, path)
		return
	}

	// Yunikorn API proxy
	if strings.HasPrefix(path, "/api/ws") {
		yunikornProxyHandler(w, r, path)
		return
	}

	// 404 for other paths
	http.NotFound(w, r)
}

func yunikornProxyHandler(w http.ResponseWriter, r *http.Request, path string) {
	apiPath := strings.TrimPrefix(path, "/api")
	targetURL := yunikornAPI + apiPath

	proxyRequest(w, r, targetURL, "Yunikorn")
}

func opensearchProxyHandler(w http.ResponseWriter, r *http.Request, path string) {
	// Extract OpenSearch API path (remove /api/opensearch prefix)
	apiPath := strings.TrimPrefix(path, "/api/opensearch")

	// Use the port-forwarded OpenSearch endpoint
	// Port-forward is set up at http://localhost:9200
	targetURL := "http://localhost:9200" + apiPath

	// Read request body for POST requests
	var bodyReader io.Reader
	if r.Method == "POST" {
		requestBody, _ := io.ReadAll(r.Body)
		bodyReader = strings.NewReader(string(requestBody))
	}

	// Create proxy request
	proxyReq, err := http.NewRequest(r.Method, targetURL, bodyReader)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create request: %v", err), http.StatusInternalServerError)
		return
	}

	// Copy headers
	for k, v := range r.Header {
		if k != "Origin" && k != "Host" {
			proxyReq.Header[k] = v
		}
	}
	if proxyReq.Header.Get("Content-Type") == "" {
		proxyReq.Header.Set("Content-Type", "application/json")
	}

	// Execute request with timeout
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(proxyReq)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to reach OpenSearch: %v (is port-forward running on :9200?)", err), http.StatusBadGateway)
		log.Printf("Failed to reach OpenSearch at %s: %v", targetURL, err)
		return
	}
	defer resp.Body.Close()

	// Copy response headers
	for k, v := range resp.Header {
		w.Header()[k] = v
	}

	// Set CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Copy status code and body
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)

	log.Printf("Proxied OpenSearch: %s -> %s (status: %d)", apiPath, targetURL, resp.StatusCode)
}

type openSearchServiceInfo struct {
	name      string
	namespace string
}

func getOpenSearchServiceInfo() openSearchServiceInfo {
	// Common OpenSearch service names and namespaces
	services := []string{
		"opensearch",
		"opensearch-service",
		"opensearch-cluster-master",
	}
	namespaces := []string{
		"opensearch",
		"default",
		"logging",
		"observability",
	}

	for _, ns := range namespaces {
		// Check if namespace exists and has service
		for _, svc := range services {
			cmd := exec.Command("kubectl", "get", "svc", svc, "-n", ns, "-o", "jsonpath={.metadata.name}")
			output, err := cmd.CombinedOutput()
			if err == nil && len(output) > 0 {
				svcName := strings.TrimSpace(string(output))
				if svcName == svc {
					log.Printf("Found OpenSearch service: %s.%s", svc, ns)
					return openSearchServiceInfo{name: svc, namespace: ns}
				}
			}
		}
	}

	return openSearchServiceInfo{}
}

func k8sProxyHandler(w http.ResponseWriter, r *http.Request, path string) {
	// Get the full URL path including query string
	fullPath := r.URL.RequestURI()
	apiPath := strings.TrimPrefix(fullPath, "/api")

	// Parse query parameters
	queryParams := ""
	if strings.Contains(apiPath, "?") {
		parts := strings.Split(apiPath, "?")
		apiPath = parts[0]
		queryParams = parts[1]
	}

	// Handle proxy requests (for metrics endpoints from pods)
	if strings.Contains(apiPath, "/proxy/") {
		proxyParts := strings.Split(apiPath, "/proxy/")
		if len(proxyParts) >= 2 {
			podName := strings.Split(proxyParts[0], "/")[6] // Extract pod name from "namespaces/{ns}/pods/{podname}"
			metricsPath := "/" + proxyParts[1]
			namespace := "default"

			// Use kubectl exec to get metrics from the pod
			// Support both /metrics/driver/prometheus/ and /metrics/executors/prometheus/ endpoints
			proxyPath := metricsPath
			if strings.HasSuffix(metricsPath, "/") {
				proxyPath = strings.TrimSuffix(metricsPath, "/")
			}
			cmd := exec.Command("kubectl", "exec", "-n", namespace, podName, "--", "curl", "-s", "http://localhost:4040"+proxyPath)
			output, err := cmd.CombinedOutput()
			if err != nil {
				http.Error(w, fmt.Sprintf("Failed to get metrics: %v", err), http.StatusBadGateway)
				log.Printf("Error getting metrics from pod %s: %v", podName, err)
				return
			}

			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Write(output)
			log.Printf("Proxied metrics from pod %s: %s", podName, metricsPath)
			return
		}
	}

	// Handle regular resource requests (list pods)
	if strings.Contains(apiPath, "namespaces/") && strings.Contains(apiPath, "/pods") {
		// Extract namespace and pod name
		namespaceParts := strings.Split(apiPath, "/")
		var namespace string
		for i, part := range namespaceParts {
			if part == "namespaces" && i+1 < len(namespaceParts) {
				namespace = namespaceParts[i+1]
				break
			}
		}

		log.Printf("Processing pod list request: namespace=%s, queryParams=%s", namespace, queryParams)

		// Build label selector from query params - decode URL encoding
		labelSelector := ""
		if queryParams != "" && strings.Contains(queryParams, "labelSelector=") {
			for _, param := range strings.Split(queryParams, "&") {
				if strings.HasPrefix(param, "labelSelector=") {
					labelSelector = strings.TrimPrefix(param, "labelSelector=")
					// URL decode: %2C -> , %3D -> = etc.
					labelSelector = strings.ReplaceAll(labelSelector, "%2C", ",")
					labelSelector = strings.ReplaceAll(labelSelector, "%3D", "=")
					labelSelector = strings.ReplaceAll(labelSelector, "%2B", "+")
					log.Printf("Extracted label selector: %s", labelSelector)
					break
				}
			}
		}

		args := []string{"get", "pods", "-n", namespace, "-o", "json"}
		if labelSelector != "" {
			args = append(args, "-l", labelSelector)
			log.Printf("Using label selector: %s", labelSelector)
		}

		cmd := exec.Command("kubectl", args...)
		output, err := cmd.CombinedOutput()
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to reach Kubernetes API: %v\nOutput: %s", err, string(output)), http.StatusBadGateway)
			log.Printf("Error reaching Kubernetes API: %v", err)
			return
		}

		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")
		w.Write(output)
		log.Printf("Proxied K8s: %s (items: %d)", path, strings.Count(string(output), `"kind": "Pod"`))
		return
	}

	// Default error for unhandled paths
	http.Error(w, "Unsupported Kubernetes API path", http.StatusNotFound)
	log.Printf("Unhandled K8s path: %s", path)
}

func proxyRequest(w http.ResponseWriter, r *http.Request, targetURL string, service string) {
	// Create new request
	proxyReq, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		http.Error(w, "Failed to create proxy request", http.StatusInternalServerError)
		log.Printf("Error creating proxy request: %v", err)
		return
	}

	// Copy headers
	for k, v := range r.Header {
		if k != "Origin" {
			proxyReq.Header[k] = v
		}
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(proxyReq)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to reach %s API", service), http.StatusBadGateway)
		log.Printf("Error reaching %s: %v", service, err)
		return
	}
	defer resp.Body.Close()

	// Copy response headers
	for k, v := range resp.Header {
		w.Header()[k] = v
	}

	// Set CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	// Copy status code and body
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)

	log.Printf("Proxied %s: %s -> %s (status: %d)", service, r.URL.Path, targetURL, resp.StatusCode)
}

func main() {
	// Start OpenSearch port-forward in background
	go startOpenSearchPortForward()

	// Setup routes with CORS
	http.HandleFunc("/", corsMiddleware(proxyHandler))

	addr := fmt.Sprintf(":%d", uiPort)
	log.Printf("Proxy Server starting on http://localhost%s", addr)
	log.Printf("  Yunikorn UI: http://localhost%s/yunikorn-ui.html", addr)
	log.Printf("  Spark Metrics UI: http://localhost%s/spark-metrics-ui.html", addr)
	log.Printf("  OpenSearch Discovery UI: http://localhost%s/opensearch-discovery-ui.html", addr)
	log.Printf("  API Proxy: http://localhost%s/api/", addr)
	log.Printf("  Yunikorn API: %s", yunikornAPI)

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal(err)
	}
}

func startOpenSearchPortForward() {
	// Try to start port-forward for OpenSearch service
	svcInfo := getOpenSearchServiceInfo()
	if svcInfo.namespace != "" && svcInfo.name != "" {
		// Kill any existing port-forward on 9200
		exec.Command("pkill", "-f", "kubectl.*port-forward.*9200").Run()

		// Start new port-forward
		cmd := exec.Command("kubectl", "port-forward", "-n", svcInfo.namespace,
			"svc/"+svcInfo.name, "9200:9200")
		cmd.Stdout = nil
		cmd.Stderr = nil

		// Start in background
		if err := cmd.Start(); err != nil {
			log.Printf("Failed to start OpenSearch port-forward: %v", err)
			return
		}

		// Wait a moment for port-forward to establish
		time.Sleep(2 * time.Second)

		// Check if port-forward is still running
		if cmd.Process == nil {
			log.Printf("OpenSearch port-forward failed to start")
		} else {
			log.Printf("OpenSearch port-forward started on localhost:9200")
		}
	}
}
