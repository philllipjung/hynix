# Fluent Bit Logging System - Master Documentation
**Project**: Hynix Kubernetes Logging Infrastructure
**Last Updated**: 2026-02-02
**Version**: 2.0

---

## Quick Links

| Document | Location | Description |
|----------|----------|-------------|
| **📖 Full Documentation** | [docs/index.md](docs/index.md) | Complete system guide |
| **🔧 Configuration** | [fluent-bit/](fluent-bit/) | Active configuration files |

---

## Documentation Structure

```
/root/hynix/
├── README.md                    (this file)
├── fluent-bit/                  # Active configuration
│   ├── fluent-bit-k8s.yaml      # K8s DaemonSet config
│   └── parsers.conf             # Parser definitions
│
└── docs/                        # All documentation
    ├── index.md                 # Main guide
    ├── guides/                  # Component guides
    │   ├── fluent-bit.md
    │   ├── spark-operator.md
    │   ├── kubelet.md
    │   └── kubernetes-events.md
    ├── reports/                 # Analysis reports
    │   ├── log-categorization.md
    │   ├── spark-operator-resolution.md
    │   └── config-validation.md
    └── reference/               # Reference configs
        ├── fluent-bit-k8s-config.yaml
        └── parsers.conf
```

---

## Quick Start

### 1. View System Overview
```bash
cat /root/hynix/docs/index.md
```

### 2. Check Configuration Status
```bash
# K8s Fluent Bit
su - philip -c "minikube kubectl -- -n logging get pods -l app=fluent-bit"

# Host Fluent Bit
systemctl status fluent-bit-host.service
```

### 3. Query Logs in OpenSearch
```bash
# Count all logs
curl -s "http://192.168.201.152:9200/unified-logs/_count" | jq '.'

# Get log breakdown
curl -s "http://192.168.201.152:9200/unified-logs/_search" \
  -H 'Content-Type: application/json' \
  -d '{"size":0,"aggs":{"ns":{"terms":{"field":"kubernetes.namespace_name.keyword"}}}}' | jq '.aggregations'
```

---

## Component Guides

| Component | Guide | Status |
|-----------|-------|--------|
| **Fluent Bit** | [docs/guides/fluent-bit.md](docs/guides/fluent-bit.md) | ✅ Running |
| **Spark Operator** | [docs/guides/spark-operator.md](docs/guides/spark-operator.md) | ✅ Collecting logs |
| **Kubelet** | [docs/guides/kubelet.md](docs/guides/kubelet.md) | ✅ Collecting logs |
| **K8s Events** | [docs/guides/kubernetes-events.md](docs/guides/kubernetes-events.md) | ✅ Collecting events |

---

## System Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Kubernetes Cluster (minikube)            │
├─────────────────────────────────────────────────────────────┤
│  ┌───────────────────────────────────────────────────────┐  │
│  │  K8s Fluent Bit DaemonSet                            │  │
│  │  - Collects: Container logs, K8s events              │  │
│  │  - Config: fluent-bit/fluent-bit-k8s.yaml            │  │
│  └───────────────────────────────────────────────────────┘  │
│                          │                                  │
│  ┌───────────────────────────────────────────────────────┐  │
│  │  Components:                                         │  │
│  │  - Spark Operator (logging)                          │  │
│  │  - Spark Driver/Executor pods (default)              │  │
│  │  - Yunikorn Scheduler (yunikorn)                     │  │
│  │  - Kube-System components (kube-system)              │  │
│  └───────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│                      OpenSearch                             │
│              192.168.201.152:9200                           │
│              Index: unified-logs                            │
└─────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────┐
│                      Linux Host                              │
├─────────────────────────────────────────────────────────────┤
│  ┌───────────────────────────────────────────────────────┐  │
│  │  Host Fluent Bit Service                             │  │
│  │  - Collects: Kubelet, microservice, kernel logs      │  │
│  │  - Config: /etc/fluent-bit/fluent-bit.conf           │  │
│  └───────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

---

## Log Collection Summary

| Log Type | Source | Method | Count | Status |
|----------|--------|--------|-------|--------|
| **Container Logs** | /var/log/containers/*.log | K8s Fluent Bit | 149K+ | ✅ Active |
| **K8s Events** | Kubernetes API | K8s Fluent Bit | 47 | ✅ Active |
| **Kubelet Logs** | minikube systemd | Host Fluent Bit | 101+ | ✅ Active |
| **Kernel Logs** | /var/log/kern.log | Host Fluent Bit | 44 | ✅ Active |
| **Microservice** | /root/hynix/server.log | Host Fluent Bit | 8 | ✅ Active |

---

## Configuration Files

### Active Files
```bash
/root/hynix/fluent-bit/
├── fluent-bit-k8s.yaml      # K8s DaemonSet (apply to cluster)
├── fluent-bit-host.conf     # Host service (copy to /etc/fluent-bit/)
└── parsers.conf             # Parser definitions
```

### Apply Configuration
```bash
# K8s Fluent Bit
cp /root/hynix/fluent-bit/fluent-bit-k8s.yaml /tmp/
chown philip:philip /tmp/fluent-bit-k8s.yaml
su - philip -c "minikube kubectl -- apply -f /tmp/fluent-bit-k8s.yaml"

# Host Fluent Bit
cp /root/hynix/fluent-bit/fluent-bit-host.conf /etc/fluent-bit/fluent-bit.conf
systemctl restart fluent-bit-host.service
```

---

## Query Examples

### Spark Operator Logs
```bash
curl -s "http://192.168.201.152:9200/unified-logs/_search" \
  -H 'Content-Type: application/json' \
  -d '{"query":{"wildcard":{"kubernetes.pod_name.keyword":"*spark-operator*"}}}'
```

### Kubelet Logs
```bash
curl -s "http://192.168.201.152:9200/unified-logs/_search" \
  -H 'Content-Type: application/json' \
  -d '{"query":{"term":{"log_type.keyword":"kubelet"}}}'
```

### Kubernetes Events
```bash
curl -s "http://192.168.201.152:9200/unified-logs/_search" \
  -H 'Content-Type: application/json' \
  -d '{"query":{"term":{"log_type.keyword":"kubernetes_event"}}}'
```

### By Namespace
```bash
curl -s "http://192.168.201.152:9200/unified-logs/_search" \
  -H 'Content-Type: application/json' \
  -d '{
    "query": {"term": {"kubernetes.namespace_name.keyword": "default"}}
  }'
```

---

## Troubleshooting

### Fluent Bit Not Collecting Logs
```bash
# Check K8s Fluent Bit
su - philip -c "minikube kubectl -- -n logging logs fluent-bit-xxx --tail=50"

# Check Host Fluent Bit
journalctl -u fluent-bit-host.service -f

# Check OpenSearch connection
curl -s "http://192.168.201.152:9200/_cluster/health"
```

### Kubelet Logs Missing
```bash
# Run pull script manually
/usr/local/bin/pull-kubelet-logs.sh

# Check log file
cat /var/log/kubelet/kubelet.log | tail -20

# Check cron job
crontab -l | grep kubelet
```

### Spark Operator Logs Not Appearing
```bash
# Check if watching file
su - philip -c "minikube kubectl -- -n logging logs fluent-bit-xxx" | grep spark-operator

# Verify pod exists
su - philip -c "minikube ssh 'ls -la /var/log/containers/*spark-operator*'"
```

---

## Recent Changes

### 2026-02-02
- ✅ Added kubelet log collection via systemd
- ✅ Fixed Spark Operator log collection (clean slate)
- ✅ Reorganized documentation structure
- ✅ Applied updated Fluent Bit K8s config

---

## Documentation Index

| Category | Document | Path |
|----------|----------|------|
| **Main** | System Guide | [docs/index.md](docs/index.md) |
| **Guides** | Fluent Bit | [docs/guides/fluent-bit.md](docs/guides/fluent-bit.md) |
| **Guides** | Spark Operator | [docs/guides/spark-operator.md](docs/guides/spark-operator.md) |
| **Guides** | Kubelet | [docs/guides/kubelet.md](docs/guides/kubelet.md) |
| **Guides** | K8s Events | [docs/guides/kubernetes-events.md](docs/guides/kubernetes-events.md) |
| **Reports** | Log Categorization | [docs/reports/log-categorization.md](docs/reports/log-categorization.md) |
| **Reports** | Issue Resolution | [docs/reports/spark-operator-resolution.md](docs/reports/spark-operator-resolution.md) |
| **Reports** | Config Validation | [docs/reports/config-validation.md](docs/reports/config-validation.md) |

---

## Support

### Quick Commands
```bash
# View all documentation
ls -la /root/hynix/docs/

# Find specific guide
find /root/hynix/docs -name "*.md"

# Check OpenSearch
curl -s "http://192.168.201.152:9200/_cat/indices?v"

# Check K8s pods
su - philip -c "minikube kubectl -- get pods --all-namespaces"
```

---

**Version**: 2.0
**Maintained By**: Data Engineering Team
**Last Update**: 2026-02-02
