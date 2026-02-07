# Apache Yunikorn Scheduler Integration

This guide covers the setup, configuration, and usage of Apache Yunikorn scheduler for gang scheduling on Kubernetes.

## Overview

Apache Yunikorn is a standalone resource scheduler for Kubernetes. It provides:
- **Gang Scheduling**: All-or-nothing scheduling for Spark executors
- **Queue-based Resource Management**: Hierarchical queues for resource allocation
- **Fine-grained Resource Control**: CPU, memory, and GPU resource management
- **Spark Native Integration**: Works seamlessly with Spark Operator

## Architecture

```
┌──────────────────────────────────────────────────────────────┐
│                    Spark Application                         │
│  ┌──────────────┐         ┌──────────────────────────────┐  │
│  │ Spark Driver │────────▶│ Spark Executors (Gang Sched) │  │
│  └──────────────┘         └──────────────────────────────┘  │
└──────────────────────────────────────────────────────────────┘
           │                           │
           ▼                           ▼
┌──────────────────────────────────────────────────────────────┐
│              Yunikorn Scheduler (Core)                       │
│  ┌───────────────┐  ┌───────────────┐  ┌─────────────────┐  │
│  │   Scheduler   │  │   Placement   │  │  State Store    │  │
│  │   Engine      │  │   Manager     │  │  (Config/State) │  │
│  └───────────────┘  └───────────────┘  └─────────────────┘  │
└──────────────────────────────────────────────────────────────┘
           │                           │
           ▼                           ▼
┌──────────────────────────────────────────────────────────────┐
│                   Kubernetes Cluster                         │
│  ┌──────────────────────────────────────────────────────┐   │
│  │   Worker Nodes (CPU, Memory, GPU Resources)          │   │
│  └──────────────────────────────────────────────────────┘   │
└──────────────────────────────────────────────────────────────┘
```

## Installation

### Prerequisites

- Kubernetes cluster (v1.20+)
- kubectl configured
- Spark Operator installed (optional, for Spark workloads)

### Step 1: Install Yunikorn Scheduler

Install Yunikorn in the default namespace:

```bash
kubectl apply -f https://raw.githubusercontent.com/apache/yunikorn-k8shim/main/deploy/yunikorn-rbac.yaml
kubectl apply -f https://raw.githubusercontent.com/apache/yunikorn-k8shim/main/deploy/yunikorn-config.yaml
kubectl apply -f https://raw.githubusercontent.com/apache/yunikorn-k8shim/main/deploy/yunikorn-deployment.yaml
```

Or install via Helm:

```bash
helm repo add yunikorn https://apache.github.io/yunikorn-k8shim
helm install yunikorn yunikorn/yunikorn -n default
```

### Step 2: Verify Installation

Check that Yunikorn pods are running:

```bash
kubectl get pods -n default -l app=yunikorn
```

Expected output:
```
NAME                                READY   STATUS    RESTARTS   AGE
yunikorn-scheduler-xxxxxxxxx-xxxxx   2/2     Running   0          1m
```

The pod should have two containers:
- **yunikorn-scheduler-k8s**: Core scheduler shim
- **yunikorn-scheduler-web**: Web UI and REST API

### Step 3: Configure Nodes for Yunikorn

Enable Yunikorn scheduling on nodes:

```bash
# Label nodes to allow Yunikorn scheduling
kubectl label nodes --all yunikorn.scheduler=enabled

# Verify labels
kubectl get nodes --show-labels | grep yunikorn
```

## Configuration

### Default Queue Configuration

Yunikorn uses a configuration map for queue definitions:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: yunikorn-configs
  namespace: default
data:
  queues.yaml: |
    partitions:
      - name: default
        queues:
          - name: root
            submitacl: '*'
            queues:
              - name: default
                resources:
                  guaranteed:
                    memory: 100000
                    vcore: 10000
                  max:
                    memory: 900000
                    vcore: 90000
              - name: min
                resources:
                  guaranteed:
                    memory: 100000
                    vcore: 10000
                  max:
                    memory: 900000
                    vcore: 90000
```

Key settings:
- **submitacl**: `*` allows all users to submit applications
- **guaranteed**: Minimum resources guaranteed to the queue
- **max**: Maximum resources the queue can use

## Spark Integration

### Gang Scheduling Configuration

To enable gang scheduling for Spark applications:

1. Add the `batchScheduler` field to your SparkApplication:

```yaml
apiVersion: sparkoperator.k8s.io/v1beta2
kind: SparkApplication
metadata:
  name: spark-app-with-yunikorn
spec:
  batchScheduler: yunikorn
  batchSchedulerOptions:
    queue: min
  driver:
    coreRequest: 1
    coreLimit: 1
    memory: "1g"
  executor:
    cores: 2
    instances: 3
    memory: "2g"
```

2. Configure Spark settings:

```yaml
sparkConf:
  spark.app.name: my-spark-app
  spark.dynamicAllocation.enabled: "false"
  spark.dynamicAllocation.shuffleTracking.enabled: "false"
```

**Important**: Disable dynamic allocation when using gang scheduling. Dynamic allocation conflicts with gang scheduling as both try to manage executor counts.

### Gang Scheduling Benefits

- **All-or-Nothing**: Either all executors are placed, or none are placed
- **Prevents Partial Failures**: Avoids scenarios where some executors start but others wait indefinitely
- **Predictable Resource Usage**: Guaranteed resources for the entire application
- **Queue-based Isolation**: Different teams can use different queues

### Example: Full SparkApplication with Yunikorn

```yaml
apiVersion: sparkoperator.k8s.io/v1beta2
kind: SparkApplication
metadata:
  name: spark-pi-yunikorn
  namespace: default
spec:
  type: Scala
  batchScheduler: yunikorn
  batchSchedulerOptions:
    queue: min
  mode: cluster
  image: "gcr.io/spark-operator/spark:v3.5.0"
  imagePullPolicy: Always
  mainClass: org.apache.spark.examples.SparkPi
  mainApplicationFile: "local:///opt/spark/examples/jars/spark-examples_2.12-3.5.0.jar"
  sparkVersion: "3.5.0"
  restartPolicy: Never
  driver:
    cores: 1
    coreLimit: "1200m"
    memory: "512m"
    serviceAccount: spark
  executor:
    cores: 1
    instances: 3
    memory: "512m"
    serviceAccount: spark
  sparkConf:
    spark.app.name: spark-pi-yunikorn
    spark.dynamicAllocation.enabled: "false"
    spark.dynamicAllocation.shuffleTracking.enabled: "false"
```

## Web UI Access

Yunikorn provides a web UI for monitoring:

```bash
# Port forward to access the UI
kubectl port-forward -n default deployment/yunikorn-scheduler 8080:8080

# Access the UI at
open http://localhost:8080
```

The UI displays:
- Cluster resource utilization
- Queue statistics
- Application status
- Allocation information
- Gang scheduling state

## Log Collection

Yunikorn logs are collected by OTEL Collector and sent to OpenSearch.

### Log Format

Yunikorn uses a tab-separated log format:

```
timestamp	level	component	message
2025-02-04 12:34:56	INFO	scheduler.core	Application app-123 submitted to queue min
```

### Searching Yunikorn Logs in OpenSearch

**Search for Yunikorn scheduler logs**:

```bash
# Port forward to OpenSearch
kubectl port-forward -n opensearch svc/opensearch 9200:9200

# Search for yunikorn-scheduler container logs
curl -X GET "https://localhost:9200/_search?pretty" -H 'Content-Type: application/json' -d'
{
  "query": {
    "wildcard": {
      "kubernetes.container_name.keyword": "*yunikorn-scheduler*"
    }
  },
  "sort": [{"@timestamp": "desc"}],
  "size": 10
}
'
```

**Search for specific application logs**:

```bash
curl -X GET "https://localhost:9200/_search?pretty" -H 'Content-Type: application/json' -d'
{
  "query": {
    "bool": {
      "must": [
        {"match": {"kubernetes.container_name": "yunikorn-scheduler-k8s"}},
        {"match": {"body": "applicationId"}}
      ]
    }
  },
  "sort": [{"@timestamp": "desc"}],
  "size": 20
}
'
```

**Search for gang scheduling events**:

```bash
curl -X GET "https://localhost:9200/_search?pretty" -H 'Content-Type: application/json' -d'
{
  "query": {
    "bool": {
      "must": [
        {"match": {"kubernetes.container_name": "yunikorn-scheduler-k8s"}},
        {"match_phrase": {"body": "gang"}}
      ]
    }
  },
  "sort": [{"@timestamp": "desc"}],
  "size": 10
}
'
```

### Available Log Fields

When querying OpenSearch, you can filter by:

| Field | Description | Example |
|-------|-------------|---------|
| `kubernetes.pod.name` | Pod name | `yunikorn-scheduler-xxxxx` |
| `kubernetes.namespace.name` | Namespace | `default` |
| `kubernetes.container.name` | Container name | `yunikorn-scheduler-k8s` |
| `kubernetes.node.name` | Node name | `minikube` |
| `body` | Log content | `Application submitted` |
| `@timestamp` | Log timestamp | `2025-02-04T12:34:56Z` |

## Troubleshooting

### Issue 1: Spark Application Stuck in "Pending" State

**Symptoms**: SparkApplication shows `SUBMITTED` but executors don't start.

**Cause**: Insufficient resources in the cluster or queue.

**Solutions**:

1. Check cluster resources:
```bash
kubectl describe nodes | grep -A 5 "Allocated resources"
```

2. Check queue capacity:
```bash
curl http://localhost:8080/ws/v1/queues
```

3. Reduce resource requests in SparkApplication:
```yaml
executor:
  cores: 1          # Reduce from 2
  memory: "1g"      # Reduce from 2g
  instances: 2      # Reduce from 3
```

### Issue 2: "No valid node found for executor" Error

**Cause**: Nodes not labeled for Yunikorn scheduling.

**Solution**:
```bash
kubectl label nodes --all yunikorn.scheduler=enabled
```

### Issue 3: Dynamic Allocation Conflicts

**Symptoms**: App fails with "Cannot use dynamic allocation with gang scheduling"

**Solution**: Disable dynamic allocation in SparkConf:
```yaml
sparkConf:
  spark.dynamicAllocation.enabled: "false"
  spark.dynamicAllocation.shuffleTracking.enabled: "false"
```

### Issue 4: Queue "rejected" Application

**Cause**: Queue doesn't exist or user doesn't have permission.

**Solutions**:

1. Verify queue exists:
```bash
curl http://localhost:8080/ws/v1/queues
```

2. Check queue ACLs in ConfigMap:
```yaml
queues:
  - name: min
    submitacl: '*'    # Allow all users
```

3. Use correct queue name:
```yaml
batchSchedulerOptions:
  queue: min    # Must match queue name in config
```

### Issue 5: "timeout waiting for task" Error

**Symptoms**: Gang scheduling timeout.

**Cause**: Not enough resources to satisfy all executors simultaneously.

**Solutions**:

1. Increase cluster resources (add nodes)
2. Reduce executor instances
3. Check for resource-hogging applications:
```bash
kubectl top pods -A
```

## Monitoring

### CLI Monitoring

**Check Yunikorn scheduler status**:

```bash
kubectl get deployment yunikorn-scheduler
kubectl logs -l app=yunikorn -c yunikorn-scheduler-k8s --tail=50
```

**Check Spark application status**:

```bash
kubectl get sparkapplications
kubectl describe sparkapplication spark-app-name
```

**Check pod status**:

```bash
kubectl get pods -l spark-role=driver,spark-app-selector=spark-app-name
kubectl get pods -l spark-role=executor,spark-app-selector=spark-app-name
```

### REST API

Yunikorn exposes REST APIs for monitoring:

```bash
# Get cluster info
curl http://localhost:8080/ws/v1/clusters

# Get queues
curl http://localhost:8080/ws/v1/queues

# Get applications
curl http://localhost:8080/ws/v1/applications

# Get specific application
curl http://localhost:8080/ws/v1/applications/application_1234567890_0001
```

### OpenSearch Dashboard

Create visualizations in OpenSearch Dashboard:

1. **Log Rate**: Count of logs per minute
   - Filter: `kubernetes.container_name: yunikorn-scheduler-k8s`
   - Aggregation: Date Histogram on `@timestamp`

2. **Queue Activity**: Distribution across queues
   - Filter: `kubernetes.container_name: yunikorn-scheduler-k8s`
   - Aggregation: Terms on `body` (queue name)

3. **Error Rate**: Error log percentage
   - Filter: `kubernetes.container_name: yunikorn-scheduler-k8s` AND `level: ERROR`
   - Aggregation: Date Histogram on `@timestamp`

## Best Practices

### 1. Queue Design

- **Separate Production/Dev**: Use different queues for different environments
- **Resource Guarantees**: Set guaranteed resources for critical queues
- **Max Limits**: Set max limits to prevent one queue from monopolizing resources

### 2. Gang Scheduling

- **Always Disable Dynamic Allocation**: Gang scheduling and dynamic allocation don't mix
- **Right-size Executor Requests**: Request what you need, not what you want
- **Test Resource Requirements**: Start small and scale up

### 3. Monitoring

- **Set Up Alerts**: Alert on high rejection rates or long pending times
- **Log Aggregation**: Use OTEL Collector to centralize logs in OpenSearch
- **Dashboards**: Create OpenSearch Dashboard visualizations

### 4. Resource Management

- **Label Nodes Properly**: Ensure all worker nodes have `yunikorn.scheduler=enabled`
- **Monitor Cluster Utilization**: Use `kubectl top nodes` and Yunikorn UI
- **Plan for Headroom**: Keep buffer resources for critical applications

## API Integration

The project's `/create` endpoint integrates with Yunikorn for gang scheduling:

**Request**:
```bash
curl -X POST http://localhost:8081/create \
  -H "Content-Type: application/json" \
  -d '{
    "provision_id": "0002_wfbm",
    "service_id": "test-00001"
  }'
```

**Response**:
```json
{
  "status": "submitted",
  "application_id": "test-00001",
  "namespace": "default"
}
```

The template `/root/hynix/template/0002_wfbm.yaml` specifies:
```yaml
batchScheduler: yunikorn
batchSchedulerOptions:
  queue: min
sparkConf:
  spark.dynamicAllocation.enabled: "false"
  spark.dynamicAllocation.shuffleTracking.enabled: "false"
```

## Performance Tuning

### Scheduler Configuration

Adjust scheduler settings in ConfigMap:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: yunikorn-configs
data:
  yunikorn.properties: |
    # Increase scheduling interval
    yunikorn.scheduler.interval=500ms

    # Increase placement timeout
    yunikorn.placement.timeout=30s

    # Enable application tags
    yunikorn.application.tags.enabled=true
```

### Resource Partitioning

Create partitions for different workloads:

```yaml
partitions:
  - name: gpu
    queues:
      - name: root
        queues:
          - name: ml-training
            properties:
            - name: resource
              value: gpu
  - name: default
    queues:
      - name: root
```

## References

- [Apache Yunikorn Documentation](https://yunikorn.apache.org/)
- [Yunikorn for Kubernetes](https://yunikorn.apache.org/docs/next/design/k8s-scheduler/)
- [Spark Integration Guide](https://yunikorn.apache.org/docs/next/integration/spark/)
- [Gang Scheduling Explained](https://yunikorn.apache.org/docs/next/design/gang_scheduling/)
