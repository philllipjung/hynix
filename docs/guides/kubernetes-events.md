# Kubernetes Events in OpenSearch - Query Guide
**Generated**: 2026-02-02
**Current Events**: 47 kubernetes events

---

## Current Status

✅ **Kubernetes events are being collected** via Fluent Bit `kubernetes_events` input plugin

**Current Statistics**:
| Metric | Count |
|--------|-------|
| Total Events | 47 |
| Normal Events | 42 |
| Warning Events | 5 |

**Top Event Reasons**:
| Reason | Count | Type |
|--------|-------|------|
| GangScheduling | 6 | Normal |
| Scheduling | 6 | Normal/Warning |
| Created | 4 | Normal |
| PodBindSuccessful | 4 | Normal |
| Pulled | 4 | Normal |
| Scheduled | 4 | Normal |
| Started | 4 | Normal |
| TaskCompleted | 4 | Normal |
| Killing | 3 | Normal |
| SparkApplicationSubmitted | 1 | Normal |
| SparkDriverRunning | 1 | Normal |
| SparkDriverCompleted | 1 | Normal |
| SparkExecutorPending | 1 | Normal |
| SparkExecutorRunning | 1 | Normal |
| SparkExecutorCompleted | 1 | Normal |

---

## Fluent Bit Configuration

**File**: `/root/hynix/fluent-bit/fluent-bit-k8s.yaml`

```ini
# INPUT: Kubernetes Events
[INPUT]
    Name              kubernetes_events
    Tag               k8s.events

# FILTER: Add log_type
[FILTER]
    Name                modify
    Match               k8s.events
    Add                 log_type     kubernetes_event

# OUTPUT: Send to OpenSearch
[OUTPUT]
    Name                opensearch
    Match               *
    Host                192.168.201.152
    Port                9200
    Index               unified-logs
```

---

## Query Examples

### 1. Get All Kubernetes Events

```bash
curl -s "http://192.168.201.152:9200/unified-logs/_search" \
  -H 'Content-Type: application/json' \
  -d '{
    "query": {
      "term": {
        "log_type.keyword": "kubernetes_event"
      }
    }
  }' | jq '.hits.hits[]'
```

### 2. Count Kubernetes Events

```bash
curl -s "http://192.168.201.152:9200/unified-logs/_count" \
  -H 'Content-Type: application/json' \
  -d '{
    "query": {
      "term": {
        "log_type.keyword": "kubernetes_event"
      }
    }
  }' | jq '.count'
```

### 3. Get Events by Type (Normal/Warning)

```bash
# Normal events only
curl -s "http://192.168.201.152:9200/unified-logs/_search" \
  -H 'Content-Type: application/json' \
  -d '{
    "query": {
      "bool": {
        "must": [
          {"term": {"log_type.keyword": "kubernetes_event"}},
          {"term": {"type.keyword": "Normal"}}
        ]
      }
    },
    "size": 10
  }'

# Warning events only
curl -s "http://192.168.201.152:9200/unified-logs/_search" \
  -H 'Content-Type: application/json' \
  -d '{
    "query": {
      "bool": {
        "must": [
          {"term": {"log_type.keyword": "kubernetes_event"}},
          {"term": {"type.keyword": "Warning"}}
        ]
      }
    },
    "size": 10
  }'
```

### 4. Get Events by Reason

```bash
# GangScheduling events
curl -s "http://192.168.201.152:9200/unified-logs/_search" \
  -H 'Content-Type: application/json' \
  -d '{
    "query": {
      "bool": {
        "must": [
          {"term": {"log_type.keyword": "kubernetes_event"}},
          {"term": {"reason.keyword": "GangScheduling"}}
        ]
      }
    }
  }'

# Spark-related events
curl -s "http://192.168.201.152:9200/unified-logs/_search" \
  -H 'Content-Type: application/json' \
  -d '{
    "query": {
      "bool": {
        "must": [
          {"term": {"log_type.keyword": "kubernetes_event"}},
          {"wildcard": {"reason.keyword": "Spark*"}}
        ]
      }
    }
  }'
```

### 5. Get Spark Application Events

```bash
# All Spark-related events
curl -s "http://192.168.201.152:9200/unified-logs/_search" \
  -H 'Content-Type: application/json' \
  -d '{
    "query": {
      "bool": {
        "must": [
          {"term": {"log_type.keyword": "kubernetes_event"}},
          {"wildcard": {"reason.keyword": "Spark*"}}
        ]
      }
    },
    "size": 20
  }' | jq '.hits.hits[] | {
      timestamp: ._source["@timestamp"],
      reason: ._source.reason,
      message: ._source.message
    }'
```

### 6. Get Events by Namespace

```bash
# Events in default namespace
curl -s "http://192.168.201.152:9200/unified-logs/_search" \
  -H 'Content-Type: application/json' \
  -d '{
    "query": {
      "bool": {
        "must": [
          {"term": {"log_type.keyword": "kubernetes_event"}},
          {"term": {"namespace_name.keyword": "default"}}
        ]
      }
    }
  }'
```

### 7. Get Events for Specific Pod

```bash
# Events for test-clean-exec-1
curl -s "http://192.168.201.152:9200/unified-logs/_search" \
  -H 'Content-Type: application/json' \
  -d '{
    "query": {
      "bool": {
        "must": [
          {"term": {"log_type.keyword": "kubernetes_event"}},
          {"wildcard": {"message.keyword": "*test-clean-exec-1*"}}
        ]
      }
    }
  }'
```

### 8. Aggregate Events by Type

```bash
curl -s "http://192.168.201.152:9200/unified-logs/_search" \
  -H 'Content-Type: application/json' \
  -d '{
    "query": {
      "term": {
        "log_type.keyword": "kubernetes_event"
      }
    },
    "size": 0,
    "aggs": {
      "event_types": {
        "terms": {
          "field": "type.keyword",
          "size": 20
        }
      },
      "event_reasons": {
        "terms": {
          "field": "reason.keyword",
          "size": 50
        }
      }
    }
  }' | jq '.aggregations'
```

### 9. Get Recent Events (Last 5 Minutes)

```bash
curl -s "http://192.168.201.152:9200/unified-logs/_search" \
  -H 'Content-Type: application/json' \
  -d '{
    "query": {
      "bool": {
        "must": [
          {"term": {"log_type.keyword": "kubernetes_event"}},
          {"range": {
            "@timestamp": {
              "gte": "now-5m"
            }
          }}
        ]
      }
    },
    "size": 20,
    "sort": [{"@timestamp": "desc"}]
  }'
```

### 10. Get Pod Lifecycle Events

```bash
# Pod creation, scheduling, and termination
curl -s "http://192.168.201.152:9200/unified-logs/_search" \
  -H 'Content-Type: application/json' \
  -d '{
    "query": {
      "bool": {
        "must": [
          {"term": {"log_type.keyword": "kubernetes_event"}},
          {"terms": {
            "reason.keyword": ["Created", "Scheduled", "Started", "Killing", "TaskCompleted"]
          }}
        ]
      }
    },
    "size": 20,
    "sort": [{"@timestamp": "desc"}]
  }'
```

---

## Sample Event Records

### GangScheduling Event
```json
{
  "@timestamp": "2026-02-02T07:11:37.000Z",
  "log_type": "kubernetes_event",
  "type": "Normal",
  "reason": "GangScheduling",
  "message": "Pod belongs to the taskGroup spark-executor, it will be scheduled as a gang member",
  "namespace_name": "default"
}
```

### SparkApplicationSubmitted Event
```json
{
  "@timestamp": "2026-02-02T07:11:36.000Z",
  "log_type": "kubernetes_event",
  "type": "Normal",
  "reason": "SparkApplicationSubmitted",
  "message": "SparkApplication test-clean was submitted",
  "namespace_name": "default"
}
```

### Scheduled Event
```json
{
  "@timestamp": "2026-02-02T07:11:37.000Z",
  "log_type": "kubernetes_event",
  "type": "Normal",
  "reason": "Scheduled",
  "message": "Successfully assigned default/test-clean-exec-1 to node minikube",
  "namespace_name": "default"
}
```

### Warning Event
```json
{
  "@timestamp": "2026-02-02T07:11:37.000Z",
  "log_type": "kubernetes_event",
  "type": "Warning",
  "reason": "Scheduling",
  "message": "Found multiple 'app-id' value in pod. { podName: test-clean-exec-1, fianlValue: test-clean, ignored: [(Label) spark-app-selector: spark-1aff9a0913ca49f798baa371429b371a] }"
}
```

---

## Event Reference

### Spark Application Lifecycle Events

| Reason | Type | Description |
|--------|------|-------------|
| SparkApplicationSubmitted | Normal | Spark application submitted to Kubernetes |
| SparkDriverRunning | Normal | Spark driver pod is running |
| SparkDriverCompleted | Normal | Spark driver completed successfully |
| SparkExecutorPending | Normal | Spark executor is pending |
| SparkExecutorRunning | Normal | Spark executor is running |
| SparkExecutorCompleted | Normal | Spark executor completed |

### Pod Scheduling Events

| Reason | Type | Description |
|--------|------|-------------|
| Scheduling | Normal/Warning | Pod is being scheduled |
| GangScheduling | Normal | Pod is part of a gang (Yunikorn) |
| Scheduled | Normal | Pod successfully scheduled to node |
| PodBindSuccessful | Normal | Pod successfully bound to node |

### Pod Lifecycle Events

| Reason | Type | Description |
|--------|------|-------------|
| Created | Normal | Pod created |
| Pulled | Normal | Container image pulled |
| Started | Normal | Container started |
| Killing | Normal | Pod is being terminated |
| TaskCompleted | Normal | Task completed |

### Other Events

| Reason | Type | Description |
|--------|------|-------------|
| IPAddressWrongReference | Warning | IP address reference issue |
| Informational | Normal | Informational message |

---

## Monitoring Queries

### Alert on Warnings
```bash
# Count warning events in last hour
curl -s "http://192.168.201.152:9200/unified-logs/_count" \
  -H 'Content-Type: application/json' \
  -d '{
    "query": {
      "bool": {
        "must": [
          {"term": {"log_type.keyword": "kubernetes_event"}},
          {"term": {"type.keyword": "Warning"}},
          {"range": {"@timestamp": {"gte": "now-1h"}}}
        ]
      }
    }
  }' | jq '.count'
```

### Monitor Spark Application Success
```bash
# Check if Spark application completed successfully
curl -s "http://192.168.201.152:9200/unified-logs/_search" \
  -H 'Content-Type: application/json' \
  -d '{
    "query": {
      "bool": {
        "must": [
          {"term": {"log_type.keyword": "kubernetes_event"}},
          {"wildcard": {"message.keyword": "*test-clean*"}},
          {"terms": {
            "reason.keyword": ["SparkDriverCompleted", "SparkExecutorCompleted"]
          }}
        ]
      }
    }
  }' | jq '.hits.total'
```

### Track Scheduling Delays
```bash
# Find pods with scheduling issues
curl -s "http://192.168.201.152:9200/unified-logs/_search" \
  -H 'Content-Type: application/json' \
  -d '{
    "query": {
      "bool": {
        "must": [
          {"term": {"log_type.keyword": "kubernetes_event"}},
          {"term": {"type.keyword": "Warning"}},
          {"wildcard": {"reason.keyword": "*Scheduling*"}}
        ]
      }
    },
    "size": 20
  }'
```

---

## Troubleshooting

### No Events Appearing

**Check Fluent Bit is running**:
```bash
kubectl -n logging get pods -l app=fluent-bit
kubectl -n logging logs fluent-bit-xxx | grep kubernetes_events
```

**Check RBAC permissions**:
```bash
kubectl get clusterrole fluent-bit -o yaml
# Should have events permissions
```

**Test event generation**:
```bash
# Create a test pod to trigger events
kubectl run test-pod --image=nginx --restart=Never
# Check events
kubectl get events --sort-by='.lastTimestamp'
# Check OpenSearch
curl -s "http://192.168.201.152:9200/unified-logs/_search" \
  -H 'Content-Type: application/json' \
  -d '{"query": {"wildcard": {"message.keyword": "*test-pod*"}}}'
```

### Missing Involved Object Information

Some events may not have full `involved_object` details. This is a limitation of the Fluent Bit kubernetes_events plugin.

---

## Configuration Reference

### Fluent Bit Kubernetes Events Plugin

**Plugin**: `kubernetes_events`

**Parameters**:
| Parameter | Default | Description |
|-----------|---------|-------------|
| Tag | k8s.events | Tag for matching records |
| URL | (none) | Kubernetes API URL |
| CA_File | (none) | CA certificate path |
| Token_File | (none) | Service account token path |
| Kubelet_Port | 10250 | Kubelet port |
| Kubelet_Host | (none) | Kubelet host |
| Kubelet_Cert | (none) | Kubelet certificate |
| Kubelet_Key | (none) | Kubelet key |

---

## Quick Reference Commands

```bash
# Get all kubernetes events
curl -s "http://192.168.201.152:9200/unified-logs/_search" \
  -H 'Content-Type: application/json' \
  -d '{"query": {"term": {"log_type.keyword": "kubernetes_event"}}}'

# Count events
curl -s "http://192.168.201.152:9200/unified-logs/_count" \
  -H 'Content-Type: application/json' \
  -d '{"query": {"term": {"log_type.keyword": "kubernetes_event"}}}' | jq '.count'

# Get Spark events
curl -s "http://192.168.201.152:9200/unified-logs/_search" \
  -H 'Content-Type: application/json' \
  -d '{"query": {"bool": {"must": [{"term": {"log_type.keyword": "kubernetes_event"}}, {"wildcard": {"reason.keyword": "Spark*"}}]}}}'

# Get warnings
curl -s "http://192.168.201.152:9200/unified-logs/_search" \
  -H 'Content-Type: application/json' \
  -d '{"query": {"bool": {"must": [{"term": {"log_type.keyword": "kubernetes_event"}}, {"term": {"type.keyword": "Warning"}}]}}}'

# Get scheduling events
curl -s "http://192.168.201.152:9200/unified-logs/_search" \
  -H 'Content-Type: application/json' \
  -d '{"query": {"bool": {"must": [{"term": {"log_type.keyword": "kubernetes_event"}}, {"wildcard": {"reason.keyword": "*Scheduling*"}}]}}}'
```

---

## Summary

✅ **Kubernetes events are being collected** in OpenSearch
📊 **Current**: 47 events from recent test-clean job
🔍 **Queries**: Easy to search and filter by type, reason, namespace
⚡ **Real-time**: Events appear within seconds of occurrence

**Next Steps**:
1. Create OpenSearch Dashboards for event visualization
2. Set up alerts for Warning events
3. Create event aggregation rules for monitoring
