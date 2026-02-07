# Prometheus Setup for Spark Operator and Yunikorn

## Overview

Prometheus has been deployed to collect metrics from:
- **Spark Operator**: Spark application metrics
- **Yunikorn Scheduler**: Gang scheduling and resource allocation metrics

## Components Deployed

### 1. Prometheus Resources

```bash
# ConfigMap with scraping configuration
kubectl get configmap prometheus-config -n default

# Deployment
kubectl get deployment prometheus -n default

# Service
kubectl get svc prometheus -n default

# ServiceAccount & RBAC
kubectl get serviceaccount prometheus -n default
kubectl get clusterrole prometheus
kubectl get clusterrolebinding prometheus
```

### 2. Files Created

| File | Description |
|------|-------------|
| `prometheus-config.yaml` | Prometheus scraping configuration |
| `prometheus-deployment.yaml` | Deployment and Service manifests |
| `prometheus-rbac.yaml` | ServiceAccount, ClusterRole, ClusterRoleBinding |
| `prometheus-forward.sh` | Port forwarding script |

## Accessing Prometheus UI

### Start Port Forward

```bash
cd /root/hynix
./prometheus-forward.sh
```

Or manually:

```bash
kubectl port-forward -n default svc/prometheus 9090:9090
```

Then open in browser: **http://localhost:9090**

## Scrape Targets

### 1. Yunikorn Scheduler
- **Job Name**: `yunikorn-scheduler`
- **Port**: `9080`
- **Metrics Path**: `/metrics`
- **Scrape Interval**: `15s`

**Key Metrics**:
- `yunikorn_scheduler_allocated_containers`
- `yunikorn_scheduler_allocated_memory`
- `yunikorn_scheduler_allocated_vcore`
- `yunikorn_scheduler_total_applications`
- `yunikorn_scheduler_queue_available_memory`
- `yunikorn_scheduler_queue_available_vcore`

### 2. Spark Operator
- **Job Name**: `spark-operator`
- **Port**: `10254`
- **Metrics Path**: `/metrics`
- **Scrape Interval**: `15s`

**Key Metrics**:
- `spark_operator_applications_total`
- `spark_operator_spark_applications_running`
- `spark_operator_spark_applications_completed`
- `spark_operator_spark_applications_failed`

## Verify Metrics Collection

### Check Targets in Prometheus UI

1. Go to: **http://localhost:9090/targets**
2. Look for:
   - `yunikorn-scheduler` (should be UP)
   - `spark-operator` (should be UP)

### Query Metrics

#### Yunikorn Metrics

```promql
# Total applications in queue
yunikorn_scheduler_total_applications

# Allocated resources
yunikorn_scheduler_allocated_memory
yunikorn_scheduler_allocated_vcore

# Queue metrics
yunikorn_scheduler_queue_available_memory{queue="root.default"}
yunikorn_scheduler_queue_available_vcore{queue="root.default"}
```

#### Spark Operator Metrics

```promql
# Running applications
spark_operator_spark_applications_running

# Application completion rate
rate(spark_operator_applications_total[5m])

# Failed applications
spark_operator_spark_applications_failed
```

## Useful Prometheus Queries

### Resource Usage by Queue

```promql
# Memory usage by queue
yunikorn_scheduler_queue_allocated_memory{queue="root.max"}

# CPU usage by queue
yunikorn_scheduler_queue_allocated_vcore{queue="root.max"}
```

### Application Status

```promql
# Running Spark applications
count(spark_operator_spark_applications_running > 0)

# Application failures per hour
rate(spark_operator_spark_applications_failed[1h]) * 3600
```

### Gang Scheduling

```promql
# Active task groups
yunikorn_scheduler_total_partitions

# Aligned applications
yunikorn_scheduler_aligned_applications
```

## Grafana Dashboard Integration

To create a Grafana dashboard:

1. Add Prometheus as data source:
   - URL: `http://prometheus:9090`
   - Access: `Server (default)`

2. Import dashboard or create panels using the queries above

## Troubleshooting

### Check Prometheus Pod Status

```bash
kubectl get pods -n default -l app=prometheus
kubectl logs -n default -l app=prometheus --tail=50
```

### Check RBAC Permissions

```bash
kubectl auth can-i list pods --as=system:serviceaccount:default:prometheus -n default
```

### Check Target Health

```bash
# Port forward first
kubectl port-forward -n default svc/prometheus 9090:9090

# Check targets API
curl http://localhost:9090/api/v1/targets | jq '.data.activeTargets[] | {job, health, lastError}'
```

### Restart Prometheus

```bash
kubectl delete pod -n default -l app=prometheus
```

## Maintenance

### Update Configuration

1. Edit the ConfigMap:
```bash
kubectl edit configmap prometheus-config -n default
```

2. Restart Prometheus:
```bash
kubectl delete pod -n default -l app=prometheus
```

### Change Retention

Edit `prometheus-deployment.yaml` and add to args:
```yaml
args:
  - '--storage.tsdb.retention.time=30d'
```

### Increase Resources

Edit `prometheus-deployment.yaml`:
```yaml
resources:
  requests:
    memory: "512Mi"
    cpu: "250m"
  limits:
    memory: "1Gi"
    cpu: "1000m"
```

## Additional Resources

- [Prometheus Documentation](https://prometheus.io/docs/)
- [Spark Operator Monitoring](https://github.com/GoogleCloudPlatform/spark-on-k8s-operator#monitoring)
- [Yunikorn Metrics](https://yunikorn.apache.org/docs/monitoring/rest_api/)
