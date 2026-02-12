# OpenSearch Log Discovery Dashboard

A web-based dashboard for searching and analyzing logs stored in OpenSearch. This dashboard provides a Discovery-style interface similar to OpenSearch Dashboards, optimized for Kubernetes container logs.

## Table of Contents

- [Overview](#overview)
- [Architecture](#architecture)
- [Features](#features)
- [Prerequisites](#prerequisites)
- [Installation](#installation)
- [Configuration](#configuration)
- [Usage](#usage)
- [API Reference](#api-reference)
- [Troubleshooting](#troubleshooting)

---

## Overview

The OpenSearch Discovery Dashboard provides:

- **Full-text log search**: Search across all container logs by keyword
- **Time range filtering**: Filter logs by time range (Last 1 hour, 7 days, 30 days, or custom)
- **Real-time results**: Display up to 10,000 log entries per search
- **Source identification**: View pod names and container sources for each log entry
- **Timestamp sorting**: Logs displayed in descending order by observed timestamp

### Key Technologies

| Component | Version | Description |
|-----------|---------|-------------|
| OpenSearch | 2.18.0 | Search and analytics engine |
| OpenSearch Dashboards | 2.18.0 | Visualization interface |
| Fluent Bit / OTEL | - | Log collector |
| Kubernetes API | - | Pod and namespace metadata |

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                      Web Browser                                │
│                  http://localhost:8083                          │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │     OpenSearch Discovery UI                               │  │
│  │     /opensearch-discovery-ui.html                         │  │
│  │  - Search by Service ID                                   │  │
│  │  - Time range filter                                      │  │
│  │  - Display log entries (max 10,000)                       │  │
│  └──────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
                              │
                              │ HTTP API
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                   Proxy Server (Go)                             │
│                       :8083                                     │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │     OpenSearch API Proxy                                  │  │
│  │     /api/opensearch/* → localhost:9200                    │  │
│  └──────────────────────────────────────────────────────────┘  │
│                                                                  │
│  Auto-discovery: kubectl get svc opensearch -n opensearch      │
│  Port-forward:  kubectl port-forward svc/opensearch 9200:9200  │
└─────────────────────────────────────────────────────────────────┘
                              │
                              │ Port-forward (localhost:9200)
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                   OpenSearch Cluster                            │
│                   opensearch.opensearch                         │
│  - Index pattern: ss4o_logs-*                                  │
│  - Log format: OTEL (OpenTelemetry)                            │
│  - Timestamp field: observedTimestamp                          │
└─────────────────────────────────────────────────────────────────┘
```

---

## Features

### 1. Keyword Search

Search logs by entering a keyword (Service ID, error message, pod name, etc.):

```json
{
  "query": {
    "bool": {
      "must": [
        {
          "wildcard": {
            "body": "*00001*"
          }
        }
      ]
    }
  }
}
```

### 2. Time Range Filtering

Filter logs by time range:

- **Last 1 hour**: `now-1h`
- **Last 7 days**: `now-7d`
- **Last 30 days**: `now-30d`
- **Custom**: Specify exact date range

Uses `observedTimestamp` field for accurate time filtering:

```json
{
  "range": {
    "observedTimestamp": {
      "gte": "2026-02-03T11:56:48.216Z",
      "lte": "2026-02-10T11:56:48.216Z"
    }
  }
}
```

### 3. Index Pattern

Logs are stored in OpenSearch with index pattern: `ss4o_logs-*`

Example indices:
- `ss4o_logs-default-namespace`
- `ss4o_logs-kube-system`
- `ss4o_logs-logging`

### 4. Log Entry Display

Each log entry shows:

| Field | Description | Example |
|-------|-------------|---------|
| Timestamp | Log creation time | `2026-02-10T11:15:11.621Z` |
| Pod Name | Source pod | `yunikorn-scheduler-756c444b87-gq9w6` |
| Namespace | Kubernetes namespace | `default` |
| Log Message | Actual log content | `2026-02-10T11:15:11.621477462...` |

---

## Prerequisites

### Required

- **Kubernetes cluster** with kubectl configured
- **OpenSearch** deployed in Kubernetes
- **Go 1.21+** (for proxy server)
- **Port 8083** available for proxy server
- **Port 9200** available for OpenSearch port-forward

### OpenSearch Installation

OpenSearch must be installed in Kubernetes without authentication:

```bash
# Example: Install OpenSearch operator
kubectl apply -f https://opensearch.org/samples/opensearch-operator.yaml

# Create OpenSearch cluster
kubectl create namespace opensearch
kubectl apply -f opensearch-cluster.yaml
```

**Resource Requirements:**

| Component | CPU | Memory |
|-----------|-----|--------|
| OpenSearch Node | 2+ cores | 4Gi+ |
| OpenSearch Dashboards | 1 core | 1Gi |

---

## Installation

### 1. Clone Repository

```bash
git clone https://github.com/your-org/hynix.git
cd hynix
```

### 2. Deploy OpenSearch

```bash
# Create namespace
kubectl create namespace opensearch

# Deploy OpenSearch (example using Helm)
helm repo add opensearch https://opensearch-project.github.io/helm-charts/
helm install opensearch opensearch/opensearch --namespace opensearch \
  --set resources.limits.memory=4Gi \
  --set replicas=1

# Deploy Dashboards
helm install opensearch-dashboards opensearch/opensearch-dashboards --namespace opensearch
```

### 3. Start Proxy Server

```bash
# Run from project root
cd /root/hynix
go run cmd/proxy/main.go > proxy.log 2>&1 &

# Or build and run
go build -o proxy-server cmd/proxy/main.go
./proxy-server
```

The proxy server will:
- Auto-discover OpenSearch service
- Start port-forward to localhost:9200
- Listen on http://localhost:8083

### 4. Access Dashboard

Open in browser:

```
http://localhost:8083/opensearch-discovery-ui.html
```

---

## Configuration

### Proxy Server Configuration

File: `cmd/proxy/main.go`

```go
const (
    uiPort = 8083  // Proxy server port
)

// OpenSearch service discovery
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
```

### Dashboard Configuration

File: `docs/opensearch-discovery-ui.html`

```javascript
const OPENSEARCH_API = '/api/opensearch';

// Index pattern
const searchUrl = `${OPENSEARCH_API}/ss4o_logs-*/_search`;

// Page size (max records per search)
const pageSize = 10000;

// Query configuration
const query = {
    size: 10000,
    sort: [{ 'observedTimestamp': { order: 'desc' } }],
    query: {
        bool: {
            must: [
                {
                    wildcard: {
                        "body": "*" + serviceId + "*"
                    }
                }
            ]
        }
    }
};
```

---

## Usage

### Basic Search

1. **Open Dashboard**: Navigate to `http://localhost:8083/opensearch-discovery-ui.html`

2. **Enter Keyword**: Type your search term (e.g., Service ID, error message)

3. **Select Time Range**: Choose from dropdown:
   - Last 1 hour
   - Last 7 days
   - Last 30 days
   - Custom range

4. **Click Search**: View matching log entries

### Example Searches

| Search For | Description |
|------------|-------------|
| `00001` | Find all logs containing "00001" |
| `ERROR` | Find error logs |
| `NullPointerException` | Find specific exceptions |
| `kube-system` | Find logs from kube-system namespace |

### Understanding Results

- **Total Hits**: Number of matching records in OpenSearch
- **Log Entries**: Actual log entries displayed (max 10,000)
- **Timestamp**: Log creation time in ISO 8601 format
- **Source**: Pod or container that generated the log

### API Proxy Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/opensearch/_search` | POST | Search logs |
| `/api/opensearch/_cat/indices` | GET | List indices |
| `/api/opensearch/ss4o_logs-*/_search` | POST | Search specific index pattern |

---

## API Reference

### Search API

**Endpoint:** `POST /api/opensearch/ss4o_logs-*/_search`

**Request Body:**

```json
{
  "size": 10000,
  "sort": [{"observedTimestamp": {"order": "desc"}}],
  "query": {
    "bool": {
      "must": [
        {
          "wildcard": {
            "body": "*KEYWORD*"
          }
        }
      ],
      "filter": [
        {
          "range": {
            "observedTimestamp": {
              "gte": "2026-02-03T11:56:48.216Z",
              "lte": "2026-02-10T11:56:48.216Z"
            }
          }
        }
      ]
    }
  }
}
```

**Response:**

```json
{
  "took": 15,
  "timed_out": false,
  "hits": {
    "total": {
      "value": 5679,
      "relation": "eq"
    },
    "hits": [
      {
        "_index": "ss4o_logs-default-namespace",
        "_id": "abc123",
        "_score": 1.0,
        "_source": {
          "@timestamp": "1970-01-01T00:00:00Z",
          "observedTimestamp": "2026-02-10T11:15:11.621Z",
          "body": "2026-02-10T11:15:11.621477462+00:00 [INFO]...",
          "severity": {},
          "attributes": {
            "log.file.name": "pod-name_namespace_container-hash.log"
          }
        }
      }
    ]
  }
}
```

---

## Troubleshooting

### Issue: "Failed to reach OpenSearch"

**Symptoms:** Dashboard shows error when searching

**Solutions:**

1. **Check OpenSearch pod status:**
   ```bash
   kubectl get pods -n opensearch
   ```

2. **Check port-forward is running:**
   ```bash
   ps aux | grep port-forward | grep 9200
   ```

3. **Test OpenSearch connection:**
   ```bash
   curl http://localhost:9200/_cat/indices
   ```

4. **Restart proxy server:**
   ```bash
   pkill -f "go run cmd/proxy/main.go"
   cd /root/hynix && go run cmd/proxy/main.go > proxy.log 2>&1 &
   ```

### Issue: "Total hits: 0" or "No logs found"

**Symptoms:** Search returns 0 results when logs should exist

**Solutions:**

1. **Check index exists:**
   ```bash
   curl http://localhost:9200/_cat/indices?v | grep ss4o_logs
   ```

2. **Verify time range:** Logs may be outside selected time range

3. **Check timestamp field:** Ensure logs use `observedTimestamp`, not `@timestamp`

4. **Test query directly:**
   ```bash
   curl -X POST "http://localhost:9200/ss4o_logs-*/_search" \
     -H "Content-Type: application/json" -d '{
     "size": 1,
     "query": {"match_all": {}}
   }'
   ```

### Issue: OpenSearch Pod OOMKilled

**Symptoms:** Pod exits with code 137

**Solution:** Increase memory limit

```bash
kubectl patch deployment opensearch -n opensearch \
  -p '{"spec":{"template":{"spec":{"containers":[{"name":"opensearch","resources":{"limits":{"memory":"4Gi"}}}]}}}}'
```

### Issue: Browser Shows Old UI

**Symptoms:** Changes not visible after update

**Solutions:**

1. **Hard refresh browser:**
   - Windows/Linux: `Ctrl + Shift + R`
   - Mac: `Cmd + Shift + R`

2. **Clear browser cache:**
   - Chrome: F12 → Network tab → "Disable cache"
   - Firefox: F12 → Network tab → "Disable cache"

3. **Use incognito/private mode:**
   - Chrome/Edge: `Ctrl + Shift + N`
   - Firefox: `Ctrl + Shift + P`

### Issue: Only Seeing Logs from One Pod

**Symptoms:** Search returns multiple pods, but UI shows only one

**Solution:** This is typically due to browser caching. Hard refresh or use incognito mode.

### Issue: Port-Forward Fails

**Symptoms:** "Address already in use" error

**Solution:** Kill existing port-forward

```bash
# Kill existing port-forward on port 9200
pkill -f "kubectl.*port-forward.*9200"

# Restart proxy server (will auto-create port-forward)
cd /root/hynix && go run cmd/proxy/main.go > proxy.log 2>&1 &
```

---

## Log Format

### OTEL Log Format

Logs are stored in OpenTelemetry format:

```json
{
  "@timestamp": "1970-01-01T00:00:00Z",
  "observedTimestamp": "2026-02-10T11:15:11.621477462Z",
  "body": "2026-02-10T11:15:11.621477462+00:00 [INFO] Starting application...",
  "severity": {},
  "instrumentationScope": {
    "name": "otel-collector",
    "version": "1.0.0"
  },
  "attributes": {
    "data_stream": {
      "dataset": "default",
      "namespace": "namespace",
      "type": "record"
    },
    "log.file.name": "pod-name_hash0_hash1_container-hash.log",
    "log.file.path": "/var/log/containers/pod-name_hash0_hash1_container-hash.log",
    "log_type": "container"
  }
}
```

### Important Fields

| Field | Usage | Notes |
|-------|-------|-------|
| `observedTimestamp` | **Use this for time filtering** | Actual log creation time |
| `@timestamp` | Do not use | Often set to epoch (1970-01-01) |
| `body` | **Full-text search** | Contains actual log message |
| `attributes.log.file.name` | Source identification | Format: `pod-name_namespace_container-hash.log` |

---

## Performance

### Query Performance

| Records | Query Time | Transfer Size |
|---------|------------|---------------|
| 1,000 | ~50ms | ~500KB |
| 5,000 | ~200ms | ~2.5MB |
| 10,000 | ~400ms | ~5MB |

### Optimization Tips

1. **Use specific keywords** instead of wildcards when possible
2. **Limit time range** to reduce search scope
3. **Use page size** 10000 for most use cases
4. **Add index timestamp** to improve time-based queries

---

## Security Considerations

### Current Configuration

- **No authentication**: OpenSearch deployed without username/password
- **Local access only**: Dashboard accessible only on localhost
- **Port-forward only**: OpenSearch accessible via localhost:9200

### Production Recommendations

For production deployments:

1. **Enable OpenSearch Security:**
   ```yaml
   security:
     enabled: true
     config:
       adminPassword: "your-secure-password"
   ```

2. **Use TLS/SSL:**
   ```yaml
   plugins:
     security:
       ssl:
         http:
           enabled: true
   ```

3. **Implement RBAC:**
   ```yaml
   rbac:
     create: true
   ```

4. **Network Policies:**
   ```yaml
   networkPolicy:
     enabled: true
   ```

---

## Changelog

### v1.0.0 (2026-02-10)

**Features:**
- Initial release
- Keyword search in log bodies
- Time range filtering
- Display up to 10,000 log entries
- Auto-discovery of OpenSearch service
- Automatic port-forward management
- Use of `observedTimestamp` for accurate time filtering

**Known Issues:**
- Browser may cache old HTML - use Ctrl+Shift+R to force refresh
- Log severity field is empty object in current OTEL format
- No authentication enabled

---

## References

- [OpenSearch Documentation](https://opensearch.org/docs/)
- [OpenTelemetry Logs](https://opentelemetry.io/docs/reference/specification/logs/)
- [Kubernetes Logging](https://kubernetes.io/docs/concepts/cluster-administration/logging/)
- [Fluent Bit](https://fluentbit.io/)
