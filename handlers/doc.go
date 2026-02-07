// Package handlers provides HTTP request handlers for the Hynix microservice.
//
// Overview
//
// This package contains all HTTP request handlers for the Spark application
// management service. Handlers are organized by functionality:
//   - health.go: Health check endpoint
//   - create.go: Spark application creation endpoint
//   - reference.go: Spark configuration reference endpoint
//
// Request Flow
//
// Each handler follows a consistent pattern:
//   1. Record start time
//   2. Parse and validate request
//   3. Load templates and configuration
//   4. Process based on provision mode (enabled/disabled)
//   5. Update YAML with service labels and resource settings
//   6. Create SparkApplication CR via Kubernetes API
//   7. Log success/error and update metrics
//
// Error Handling
//
// All errors are logged and returned as JSON responses with appropriate
// HTTP status codes:
//   - 400: Bad Request (validation errors)
//   - 404: Not Found (template/config not found)
//   - 500: Internal Server Error (processing failures)
//
// Metrics
//
// All requests are tracked with Prometheus metrics:
//   - requests_total: Total requests by endpoint and status
//   - request_duration_seconds: Request duration by provision ID
//   - k8s_creation_total: Kubernetes CR creation attempts
//   - provision_mode: Provision mode (enabled/disabled)
//   - queue_selection: Selected queue for resource allocation
//
// Constants
//
// Log field keys are defined as constants for consistency:
//   - LogFieldEndpoint: API endpoint name
//   - LogFieldProvisionID: Provision identifier
//   - LogFieldServiceID: Service identifier
//   - LogFieldCategory: Job category
//   - LogFieldRegion: Geographic region
//   - LogFieldNamespace: Kubernetes namespace
//   - LogFieldResourceName: Resource name
//   - LogFieldEnabled: Provision enabled flag
//   - LogFieldReason: Reason for status
//   - LogFieldDurationMs: Request duration in milliseconds
package handlers
