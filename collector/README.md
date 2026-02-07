# OpenTelemetry Collector Configuration

This directory contains the configuration files for deploying OpenTelemetry Collector with OpenSearch exporter for log collection in Kubernetes.

## Quick Start

```bash
# Deploy OTEL Collector
./deployment.sh
```

## Files

| File | Description |
|------|-------------|
| `configmap.yaml` | OTEL Collector configuration (receivers, processors, exporters) |
| `rbac.yaml` | RBAC configuration for Kubernetes metadata access |
| `volume-patch.yaml` | JSON patch for /var/log volume mount |
| `deployment.sh` | Automated deployment script |
| `README.md` | This file |

## Configuration Details

### Receivers

**filelog**: Collects container logs from `/var/log/containers/*.log`
- Excludes fluent-bit and opentelemetry-collector logs
- Adds `log_type` attribute
- Starts reading from beginning of file

**otlp**: Receives OTLP protocol data (metrics, traces)
- gRPC endpoint: `0.0.0.0:4317`
- HTTP endpoint: `0.0.0.0:4318`

### Processors

- **batch**: Batches log records (1000 records per batch, 5s timeout)
- **k8sattributes**: Enriches logs with Kubernetes metadata (pod, namespace, node, container)
- **memory_limiter**: Prevents OOM errors (80% limit, 25% spike limit)

### Exporters

- **opensearch**: Sends logs to OpenSearch at `http://opensearch.opensearch.svc.cluster.local:9200`
- **debug**: Outputs logs to stdout for debugging

### Service Pipelines

- **logs**: filelog → memory_limiter → batch → k8sattributes → opensearch + debug
- **metrics**: otlp → memory_limiter → batch → debug
- **traces**: otlp → memory_limiter → batch → debug

## Manual Deployment

If you prefer manual deployment:

```bash
# 1. Apply RBAC
kubectl apply -f rbac.yaml

# 2. Apply ConfigMap
kubectl apply -f configmap.yaml

# 3. Install via Helm
helm repo add open-telemetry https://open-telemetry.github.io/opentelemetry-helm-charts
helm repo update

helm install otel-collector open-telemetry/opentelemetry-collector \
  --namespace default \
  --set mode=daemonset \
  --set image.repository="otel/opentelemetry-collector-contrib" \
  --set image.tag="0.122.0" \
  --set configMap.name=otel-collector-opentelemetry-collector-agent \
  --set configMap.key=relay \
  --set serviceAccount.name=otel-collector-opentelemetry-collector \
  --set serviceAccount.create=false

# 4. Patch for volume mounts
kubectl patch daemonset otel-collector-opentelemetry-collector-agent \
  -n default --type='json' -p="$(cat volume-patch.yaml)"
```

## Verification

### Check Pod Status

```bash
kubectl get pods -n default -l app.kubernetes.io/name=opentelemetry-collector
```

Expected output:
```
NAME                                                  READY   STATUS    RESTARTS   AGE
otel-collector-opentelemetry-collector-agent-xxxxx    1/1     Running   0          1m
```

### View Logs

```bash
kubectl logs -n default -l app.kubernetes.io/name=opentelemetry-collector --tail=50
```

### Check Volume Mounts

```bash
kubectl describe pod -l app.kubernetes.io/name=opentelemetry-collector | grep -A 5 "Mounts:"
```

Should include:
```
Mounts:
  /var/log from varlog (ro)
```

### Verify Logs in OpenSearch

```bash
# Port forward to OpenSearch
kubectl port-forward -n opensearch svc/opensearch 9200:9200

# Search for logs
curl -X GET "https://localhost:9200/_search?pretty" -H 'Content-Type: application/json' -d'
{
  "query": {"match_all": {}},
  "sort": [{"@timestamp": "desc"}],
  "size": 1
}
'
```

## Troubleshooting

### Issue: "finding files error: no files match the configured criteria"

**Cause**: /var/log not mounted

**Solution**:
```bash
kubectl patch daemonset otel-collector-opentelemetry-collector-agent \
  -n default --type='json' -p="$(cat volume-patch.yaml)"
```

### Issue: "pods is forbidden: User cannot list pods"

**Cause**: Missing RBAC permissions

**Solution**:
```bash
kubectl apply -f rbac.yaml
```

### Issue: No logs in OpenSearch

**Debug**:
```bash
# Check OTEL logs
kubectl logs -n default -l app.kubernetes.io/name=opentelemetry-collector

# Check if log files exist
kubectl exec -it otel-collector-opentelemetry-collector-agent-xxxxx -- ls -la /var/log/containers/

# Check OpenSearch connection
kubectl exec -it otel-collector-opentelemetry-collector-agent-xxxxx -- \
  curl http://opensearch.opensearch.svc.cluster.local:9200/_cat/indices
```

## Monitoring

The OTEL Collector exposes Prometheus metrics on port 8888:

```bash
kubectl port-forward otel-collector-opentelemetry-collector-agent-xxxxx 8888:8888
curl http://localhost:8888/metrics
```

Key metrics:
- `otelcol_receiver_accepted_log_records`: Total log records received
- `otelcol_exporter_sent_log_records`: Total log records sent to OpenSearch

## Customization

### Change OpenSearch Endpoint

Edit `configmap.yaml`:

```yaml
opensearch:
  http:
    endpoint: http://your-opensearch-endpoint:9200
    tls:
      insecure: true
```

Then apply:
```bash
kubectl apply -f configmap.yaml
kubectl rollout restart daemonset otel-collector-opentelemetry-collector-agent -n default
```

### Exclude Additional Containers

Edit `configmap.yaml`:

```yaml
receivers:
  filelog:
    exclude:
      - /var/log/containers/*fluent-bit*.log
      - /var/log/containers/*opentelemetry-collector*.log
      - /var/log/containers/*your-container*.log
```

### Adjust Batch Settings

Edit `configmap.yaml`:

```yaml
processors:
  batch:
    timeout: 10s
    send_batch_size: 10000
```

## Uninstall

```bash
# Uninstall Helm release
helm uninstall otel-collector -n default

# Delete ConfigMap
kubectl delete configmap otel-collector-opentelemetry-collector-agent -n default

# Delete RBAC
kubectl delete -f rbac.yaml
```

## References

- [OpenTelemetry Collector Documentation](https://opentelemetry.io/docs/collector/)
- [OpenSearch Exporter Documentation](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/exporter/opensearchexporter)
- [Complete Guide](../docs/guides/otel-collector.md)
