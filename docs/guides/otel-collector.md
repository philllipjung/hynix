# OpenTelemetry Collector with OpenSearch

This guide covers the setup and configuration of OpenTelemetry Collector for collecting Kubernetes logs and sending them to OpenSearch.

## Overview

The OpenTelemetry Collector (OTEL) is used as a centralized log collection agent that:
- Collects container logs from all Kubernetes pods
- Enriches logs with Kubernetes metadata (pod, namespace, container, node)
- Sends logs to OpenSearch for storage and analysis
- Runs as a DaemonSet on each node in the cluster

## Architecture

```
┌─────────────────┐     ┌──────────────────┐     ┌─────────────────┐
│   Containers    │────▶│  OTEL Collector  │────▶│   OpenSearch    │
│  (Pod Logs)     │     │   (DaemonSet)    │     │  (Log Storage)  │
└─────────────────┘     └──────────────────┘     └─────────────────┘
                              │
                              ▼
                       ┌──────────────────┐
                       │  k8s Attributes  │
                       │  (Metadata)      │
                       └──────────────────┘
```

## Installation

### Prerequisites

- Kubernetes cluster (minikube, AKS, EKS, GKE, etc.)
- OpenSearch cluster running
- kubectl configured to access the cluster

### Step 1: Install RBAC Configuration

The OTEL Collector requires RBAC permissions to access Kubernetes API for metadata enrichment:

```bash
kubectl apply -f - <<EOF
apiVersion: v1
kind: ServiceAccount
metadata:
  name: otel-collector
  namespace: default

---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: otel-collector
rules:
- apiGroups: [""]
  resources:
  - pods
  - namespaces
  - nodes
  verbs: ["get", "list", "watch"]

---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: otel-collector
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: otel-collector
subjects:
- kind: ServiceAccount
  name: otel-collector
  namespace: default
EOF
```

### Step 2: Create ConfigMap with OTEL Configuration

Save the following configuration to `otel-config.yaml`:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: otel-collector-config
  namespace: default
data:
  relay: |
    receivers:
      filelog:
        include:
          - /var/log/containers/*.log
        exclude:
          - /var/log/containers/*fluent-bit*.log
          - /var/log/containers/*opentelemetry-collector*.log
        start_at: beginning
        include_file_path: true
        include_file_name: true
        operators:
          - type: add
            field: attributes.log_type
            value: container

      otlp:
        protocols:
          grpc:
            endpoint: ${env:MY_POD_IP}:4317
          http:
            endpoint: ${env:MY_POD_IP}:4318

    processors:
      batch:
        timeout: 5s
        send_batch_size: 1000

      k8sattributes:
        auth_type: serviceAccount
        extract:
          metadata:
            - k8s.pod.name
            - k8s.namespace.name
            - k8s.pod.uid
            - k8s.node.name
            - k8s.container.name
        pod_association:
          - sources:
            - from: resource_attribute
              name: k8s.pod.uid

      memory_limiter:
        check_interval: 5s
        limit_percentage: 80
        spike_limit_percentage: 25

    exporters:
      debug:
        verbosity: normal

      opensearch:
        http:
          endpoint: http://opensearch.opensearch.svc.cluster.local:9200
          tls:
            insecure: true

    extensions:
      health_check:
        endpoint: ${env:MY_POD_IP}:13133

    service:
      extensions:
      - health_check

      pipelines:
        logs:
          receivers:
          - filelog
          - otlp
          processors:
          - memory_limiter
          - batch
          - k8sattributes
          exporters:
          - opensearch
          - debug

        metrics:
          receivers:
          - otlp
          processors:
          - memory_limiter
          - batch
          exporters:
          - debug

        traces:
          receivers:
          - otlp
          processors:
          - memory_limiter
          - batch
          exporters:
          - debug

      telemetry:
        metrics:
          readers:
          - pull:
              exporter:
                prometheus:
                  host: ${env:MY_POD_IP}
                  port: 8888
```

Apply the ConfigMap:

```bash
kubectl apply -f otel-config.yaml
```

### Step 3: Install OTEL Collector via Helm

```bash
# Add the OpenTelemetry Helm repository
helm repo add open-telemetry https://open-telemetry.github.io/opentelemetry-helm-charts
helm repo update

# Install the collector
helm install otel-collector open-telemetry/opentelemetry-collector \
  --namespace default \
  --set mode=daemonset \
  --set image.repository="otel/opentelemetry-collector-contrib" \
  --set image.tag="0.122.0" \
  --set configMap.name=otel-collector-config \
  --set configMap.key=relay \
  --set serviceAccount.name=otel-collector
```

### Step 4: Add Volume Mounts for Log Access

The OTEL Collector needs access to `/var/log` on the host node to read container logs:

```bash
kubectl patch daemonset otel-collector-opentelemetry-collector-agent \
  -n default \
  --type='json' \
  -p='[
    {
      "op": "add",
      "path": "/spec/template/spec/volumes/-",
      "value": {
        "name": "varlog",
        "hostPath": {"path": "/var/log"}
      }
    },
    {
      "op": "add",
      "path": "/spec/template/spec/containers/0/volumeMounts/-",
      "value": {
        "name": "varlog",
        "mountPath": "/var/log",
        "readOnly": true
      }
    }
  ]'
```

### Step 5: Verify Installation

Check that the OTEL Collector pods are running:

```bash
kubectl get pods -n default -l app.kubernetes.io/name=opentelemetry-collector
```

Expected output:
```
NAME                                                  READY   STATUS    RESTARTS   AGE
otel-collector-opentelemetry-collector-agent-xxxxx    1/1     Running   0          1m
```

## Configuration Explained

### Receivers

**filelog**: Reads container log files
- `include`: Pattern to match log files (`/var/log/containers/*.log`)
- `exclude`: Exclude specific containers (fluent-bit, opentelemetry-collector)
- `start_at`: Read from beginning of file (set to `end` for new logs only)
- `include_file_path`: Adds file path as an attribute
- `operators`: Adds custom attributes (e.g., `log_type`)

**otlp**: Receives OTLP protocol data (metrics, traces)
- `grpc`: gRPC endpoint on port 4317
- `http`: HTTP endpoint on port 4318

### Processors

**batch**: Batches log records for efficient transmission
- `timeout`: Maximum time to wait before sending a batch
- `send_batch_size`: Maximum number of records per batch

**k8sattributes**: Enriches logs with Kubernetes metadata
- `extract`: List of metadata fields to extract
- `pod_association`: How to associate logs with pods (via UID)

**memory_limiter**: Prevents OOM errors
- `check_interval`: How often to check memory usage
- `limit_percentage`: Memory usage threshold
- `spike_limit_percentage`: Temporary spike threshold

### Exporters

**opensearch**: Sends logs to OpenSearch
- `http.endpoint`: OpenSearch endpoint URL
- `http.tls.insecure`: Skip TLS verification (for development)

**debug**: Outputs logs to stdout for debugging
- `verbosity`: Log level (normal, detailed, etc.)

### Service Pipelines

**logs**: Pipeline for log data
- Receivers: filelog, otlp
- Processors: memory_limiter, batch, k8sattributes
- Exporters: opensearch, debug

**metrics**: Pipeline for metrics data
- Receivers: otlp
- Processors: memory_limiter, batch
- Exporters: debug

**traces**: Pipeline for trace data
- Receivers: otlp
- Processors: memory_limiter, batch
- Exporters: debug

## Troubleshooting

### Issue 1: "finding files error: no files match the configured criteria"

**Cause**: The `/var/log` directory is not mounted in the OTEL Collector pod.

**Solution**: Add volume mounts (see Step 4 in Installation)

**Verification**:
```bash
kubectl describe pod -l app.kubernetes.io/name=opentelemetry-collector | grep -A 5 "Mounts:"
```

### Issue 2: "pods is forbidden: User cannot list pods"

**Cause**: Missing RBAC permissions for k8sattributes processor.

**Solution**: Apply RBAC configuration (see Step 1 in Installation)

**Verification**:
```bash
kubectl auth can-i list pods --as=system:serviceaccount:default:otel-collector
```

### Issue 3: "failed to flush chunk" in Fluent Bit

**Cause**: OpenSearch mapping conflicts or incompatible exporter.

**Solution**: Use OTEL Collector with opensearch exporter instead of elasticsearch exporter.

**Why**: The elasticsearch exporter uses the official Elasticsearch Go client which rejects OpenSearch. Use the opensearch exporter with nested HTTP configuration.

### Issue 4: No logs appearing in OpenSearch

**Debugging Steps**:

1. Check OTEL Collector logs:
```bash
kubectl logs -l app.kubernetes.io/name=opentelemetry-collector --tail=50
```

2. Check if log files exist:
```bash
kubectl exec -it otel-collector-opentelemetry-collector-agent-xxxxx -- ls -la /var/log/containers/
```

3. Check OpenSearch connection:
```bash
kubectl exec -it otel-collector-opentelemetry-collector-agent-xxxxx -- curl http://opensearch.opensearch.svc.cluster.local:9200/_cat/indices
```

4. Enable debug exporter (already enabled in config) and check output:
```bash
kubectl logs -l app.kubernetes.io/name=opentelemetry-collector | grep -A 10 "LogRecord"
```

### Issue 5: OTEL Exporter Configuration Key Errors

**Symptoms**: Invalid key errors like `endpoint`, `endpoints`, `logs_index` not recognized.

**Cause**: Incorrect opensearch exporter configuration format.

**Solution**: Use the nested HTTP configuration format (required for OTEL v0.122.0+):
```yaml
opensearch:
  http:
    endpoint: http://opensearch.opensearch.svc.cluster.local:9200
    tls:
      insecure: true
```

## Version Compatibility

| Component | Version | Notes |
|-----------|---------|-------|
| OTEL Collector | 0.122.0 | Contrib distribution with opensearch exporter |
| OpenSearch | 2.18.0 | Tested version |
| Kubernetes | 1.29+ | Tested on minikube |
| containerd | Latest | CRI-compliant runtime |

## Upgrading from Fluent Bit

If migrating from Fluent Bit to OTEL Collector:

1. Remove Fluent Bit:
```bash
helm uninstall fluent-bit
kubectl delete configmap fluent-bit-config
```

2. Install OTEL Collector following the steps in this guide.

3. Verify log collection:
```bash
# Port forward to OpenSearch
kubectl port-forward -n opensearch svc/opensearch 9200:9200

# Search for recent logs
curl -X GET "https://localhost:9200/_search?pretty" -H 'Content-Type: application/json' -d'
{
  "query": {"match_all": {}},
  "sort": [{"@timestamp": "desc"}],
  "size": 1
}
'
```

## Performance Tuning

### Batch Size Adjustment

For high-volume log environments, adjust batch settings:

```yaml
processors:
  batch:
    timeout: 10s
    send_batch_size: 10000
```

### Memory Limits

Increase memory limits for the collector pod:

```bash
kubectl set resources daemonset otel-collector-opentelemetry-collector-agent \
  --limits=memory=512Mi \
  --requests=memory=256Mi
```

### Exclude Unnecessary Logs

Reduce log volume by excluding debug containers:

```yaml
receivers:
  filelog:
    exclude:
      - /var/log/containers/*fluent-bit*.log
      - /var/log/containers/*opentelemetry-collector*.log
      - /var/log/containers/*debug*.log
```

## Monitoring

The OTEL Collector exposes Prometheus metrics on port 8888:

```bash
kubectl port-forward otel-collector-opentelemetry-collector-agent-xxxxx 8888:8888
curl http://localhost:8888/metrics
```

Key metrics to monitor:
- `otelcol_receiver_accepted_log_records`: Total log records received
- `otelcol_exporter_sent_log_records`: Total log records sent to OpenSearch
- `otelcol_processor_batch_batch_send_size`: Batch size distribution

## References

- [OpenTelemetry Collector Documentation](https://opentelemetry.io/docs/collector/)
- [OpenSearch Exporter Documentation](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/exporter/opensearchexporter)
- [Kubernetes Attributes Processor](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/processor/k8sattributesprocessor)
