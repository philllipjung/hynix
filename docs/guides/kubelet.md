# Kubelet Log Collection - Complete Guide
**Generated**: 2026-02-02
**Status**: ✅ Active (101 logs collected)

---

## Table of Contents

1. [Overview](#overview)
2. [Architecture](#architecture)
3. [Setup Configuration](#setup-configuration)
4. [Log Format](#log-format)
5. [Query Examples](#query-examples)
6. [Troubleshooting](#troubleshooting)

---

## Overview

### What is Kubelet?

**Kubelet** is the primary Kubernetes node agent that runs on each node. It is responsible for:

- Managing pods on the node
- Communicating with the Kubernetes API server
- Managing container runtime interactions (containerd, CRI-O)
- Reporting node and pod status
- Mounting volumes
- Handling pod lifecycle events

### Why Collect Kubelet Logs?

Kubelet logs provide visibility into:
- **Pod scheduling issues** - Why pods aren't starting
- **Container runtime errors** - Communication failures with containerd
- **Volume mounting problems** - Storage attachment issues
- **Network configuration** - CNI plugin issues
- **Resource allocation** - CPU/memory pressure

---

## Architecture

### Current Setup

```
┌─────────────────────────────────────────────────────────────────┐
│                      Minikube VM                               │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  kubelet (PID 1359)                                              │
│  └─ Logs written to: systemd journal                            │
│                                                                  │
│     (minikube ssh + journalctl -u kubelet)                      │
│                  │                                               │
│                  │ Pull Script (every 5 min)                    │
│                  ▼                                               │
└──────────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Host Machine                                 │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  /var/log/kubelet/kubelet.log                                   │
│  └─ Parsed kubelet logs from minikube                          │
│                  │                                               │
│                  │ Fluent Bit tail input                        │
│                  ▼                                               │
│  fluent-bit-host.service                                        │
│  └─ Reads /var/log/kubelet/kubelet.log                         │
│                  │                                               │
│                  ▼                                               │
└──────────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────────┐
│                      OpenSearch                                 │
├─────────────────────────────────────────────────────────────────┤
│  Index: unified-logs                                            │
│  log_type: kubelet                                              │
│  log_source: host                                               │
│  Count: 101+ logs                                              │
└─────────────────────────────────────────────────────────────────┘
```

### Why This Approach?

| Challenge | Solution |
|-----------|----------|
| Kubelet runs **inside minikube**, not on host | Pull logs via minikube ssh |
| Kubelet logs in **systemd journal**, not file | Use `journalctl -u kubelet` |
| Host Fluent Bit **cannot access** minikube internal filesystem | Script pulls logs to host filesystem |
| Logs need **periodic updates** | Cron job runs every 5 minutes |

---

## Setup Configuration

### 1. Log Pull Script

**Location**: `/usr/local/bin/pull-kubelet-logs.sh`

```bash
#!/bin/bash
# Pull kubelet logs from minikube to host

LOG_DIR="/var/log/kubelet"
LOG_FILE="${LOG_DIR}/kubelet.log"

# Create log directory if not exists
mkdir -p "$LOG_DIR"

# Pull recent logs from minikube (last 100 lines)
su - philip -c "minikube ssh 'sudo journalctl -u kubelet -n 100 --no-pager --output=short'" > "$LOG_FILE.tmp" 2>/dev/null

# Only replace if successful
if [ $? -eq 0 ] && [ -s "$LOG_FILE.tmp" ]; then
    mv "$LOG_FILE.tmp" "$LOG_FILE"
    chmod 644 "$LOG_FILE"
else
    rm -f "$LOG_FILE.tmp"
fi
```

**Permissions**:
```bash
chmod +x /usr/local/bin/pull-kubelet-logs.sh
```

### 2. Cron Job

**Purpose**: Run pull script every 5 minutes

```bash
# View current crontab
crontab -l | grep pull-kubelet

# Output:
# */5 * * * * /usr/local/bin/pull-kubelet-logs.sh
```

### 3. Fluent Bit Configuration

**Location**: `/etc/fluent-bit/fluent-bit.conf`

```ini
# INPUT: Kubelet 로그 (pulled from minikube)
[INPUT]
    Name              tail
    Path              /var/log/kubelet/kubelet.log
    Tag               host.kubelet
    Parser            syslog
    Refresh_Interval  60
    Mem_Buf_Limit     10MB
    Skip_Long_Lines   On
    DB                /var/lib/fluent-bit/flb_kubelet.db
    DB.Sync           Normal
    DB.Locking        True
    Read_from_Head    False

# FILTER: Add log_type
[FILTER]
    Name                modify
    Match               host.kubelet
    Add                 log_type     kubelet
    Add                 log_source   host
```

**Parameters**:
| Parameter | Value | Purpose |
|-----------|-------|---------|
| `Path` | `/var/log/kubelet/kubelet.log` | Log file location |
| `Parser` | `syslog` | Parse systemd journal format |
| `Refresh_Interval` | 60 | Check for new logs every 60 seconds |
| `Read_from_Head` | `False` | Only read new data after start |

### 4. Fluent Bit Service

**Service**: `fluent-bit-host.service`

```bash
# Check status
systemctl status fluent-bit-host.service

# Restart service
systemctl restart fluent-bit-host.service

# View logs
journalctl -u fluent-bit-host.service -f
```

---

## Log Format

### Kubelet Log Structure

```
DATE TIME HOSTNAME SERVICE[PID]: LEVEL TIME PID SOURCE_FILE:LINE] MESSAGE
```

### Example Log

```
Feb 02 08:08:41 minikube kubelet[1359]: E0202 08:08:41.252312    1359 container_log_manager.go:263] "Failed to rotate log for container" err="failed to rotate log..."
```

### Field Breakdown

| Field | Value | Description |
|-------|-------|-------------|
| `DATE` | `Feb 02 08:08:41` | Timestamp (systemd format) |
| `HOSTNAME` | `minikube` | Node hostname |
| `SERVICE` | `kubelet` | Service name |
| `PID` | `1359` | Process ID |
| `LEVEL` | `E0202` | Log level (E=Error, I=Info, W=Warning) |
| `TIME` | `08:08:41.252312` | Go timestamp |
| `SOURCE_FILE` | `container_log_manager.go` | Kubelet source file |
| `LINE` | `263` | Line number in source |
| `MESSAGE` | JSON or text | Log message |

### Log Levels

| Level | Format | Description |
|-------|--------|-------------|
| **Info** | `I0202` | Informational messages |
| **Error** | `E0202` | Error conditions |
| **Warning** | `W0202` | Warning messages |
| **Fatal** | `F0202` | Fatal errors |

---

## Query Examples

### 1. Get All Kubelet Logs

```bash
curl -s "http://192.168.201.152:9200/unified-logs/_search" \
  -H 'Content-Type: application/json' \
  -d '{
    "query": {
      "term": {
        "log_type.keyword": "kubelet"
      }
    }
  }'
```

### 2. Count Kubelet Logs

```bash
curl -s "http://192.168.201.152:9200/unified-logs/_count" \
  -H 'Content-Type: application/json' \
  -d '{
    "query": {
      "term": {
        "log_type.keyword": "kubelet"
      }
    }
  }' | jq '.count'
```

### 3. Get Error Logs Only

```bash
curl -s "http://192.168.201.152:9200/unified-logs/_search" \
  -H 'Content-Type: application/json' \
  -d '{
    "query": {
      "bool": {
        "must": [
          {"term": {"log_type.keyword": "kubelet"}},
          {"wildcard": {"log": "*E0202*"}}
        ]
      }
    },
    "size": 20
  }'
```

### 4. Container Log Rotation Failures

```bash
curl -s "http://192.168.201.152:9200/unified-logs/_search" \
  -H 'Content-Type: application/json' \
  -d '{
    "query": {
      "bool": {
        "must": [
          {"term": {"log_type.keyword": "kubelet"}},
          {"wildcard": {"log": "*Failed to rotate*"}}
        ]
      }
    }
  }'
```

### 5. Volume Issues

```bash
curl -s "http://192.168.201.152:9200/unified-logs/_search" \
  -H 'Content-Type: application/json' \
  -d '{
    "query": {
      "bool": {
        "must": [
          {"term": {"log_type.keyword": "kubelet"}},
          {"wildcard": {"log": "*volume*"}}
        ]
      }
    }
  }'
```

### 6. Specific Source File

```bash
# Get logs from kubelet_volumes.go
curl -s "http://192.168.201.152:9200/unified-logs/_search" \
  -H 'Content-Type: application/json' \
  -d '{
    "query": {
      "bool": {
        "must": [
          {"term": {"log_type.keyword": "kubelet"}},
          {"wildcard": {"log": "*kubelet_volumes.go*"}}
        ]
      }
    }
  }'
```

### 7. Recent Logs (Last Hour)

```bash
curl -s "http://192.168.201.152:9200/unified-logs/_search" \
  -H 'Content-Type: application/json' \
  -d '{
    "query": {
      "bool": {
        "must": [
          {"term": {"log_type.keyword": "kubelet"}},
          {"range": {
            "@timestamp": {
              "gte": "now-1h"
            }
          }}
        ]
      }
    },
    "sort": [{"@timestamp": "desc"}]
  }'
```

### 8. Aggregate by Log Pattern

```bash
curl -s "http://192.168.201.152:9200/unified-logs/_search" \
  -H 'Content-Type: application/json' \
  -d '{
    "query": {
      "term": {
        "log_type.keyword": "kubelet"
      }
    },
    "size": 0,
    "aggs": {
      "log_patterns": {
        "terms": {
          "field": "log.keyword",
          "size": 50
        }
      }
    }
  }'
```

---

## Common Kubelet Log Messages

### 1. Container Log Rotation Issues

```
E0202 08:08:41.252312    1359 container_log_manager.go:263] "Failed to rotate log for container"
err="failed to rotate log \"/var/log/pods/...\": failed to reopen container log"
containerID="5617dce1b00d43022c724cbf4674aeecb0dd1ff49af14195be5e8e69fb534b81"
```

**Meaning**: Kubelet cannot rotate container logs (typically due to Docker not supporting log reopening)

**Severity**: Warning - usually not critical, but may cause disk space issues

---

### 2. Orphaned Pod Volume Cleanup

```
I0202 08:11:50.226501    1359 kubelet_volumes.go:163] "Cleaned up orphaned pod volumes dir"
podUID="ee0a637e-eac3-4189-81ca-28db7b34b49c"
path="/var/lib/kubelet/pods/ee0a637e-eac3-4189-81ca-28db7b34b49c/volumes"
```

**Meaning**: Kubelet cleaned up volumes for a deleted pod

**Severity**: Info - normal operation

---

### 3. Container Removal

```
I0202 08:11:56.926527    1359 scope.go:117] "RemoveContainer"
containerID="ec23174be0d937a1924fa5a5d044d452c04510a4a188010739cc006aec069ba1"
```

**Meaning**: Kubelet removed a container

**Severity**: Info - normal operation

---

### 4. Runtime Service Errors

```
E0202 08:11:31.342442    1359 log.go:32] "ReopenContainerLog from runtime service failed"
err="rpc error: code = Unknown desc = docker does not support reopening container log files"
```

**Meaning**: Container runtime doesn't support log reopening

**Severity**: Warning - typically not critical

---

### 5. PLEG (Pod Lifecycle Event Generator) Issues

```
E0202 08:00:00.123456    1359 pleg.go:130] "PLEG is not healthy"
```

**Meaning**: Pod lifecycle event generator is unhealthy

**Severity**: Error - may indicate node issues

---

## Kubelet Source Files

| Source File | Purpose |
|-------------|---------|
| `kubelet.go` | Main kubelet implementation |
| `kubelet_volumes.go` | Volume management |
| `container_log_manager.go` | Container log rotation |
| `log.go` | Logging utilities |
| `scope.go` | Pod/container lifecycle scope |
| `pleg.go` | Pod lifecycle event generator |
| `pod_workers.go` | Pod worker queue management |
| `network.go` | Network plugin management |
| `runtime.go` | Container runtime interface |

---

## Troubleshooting

### No Kubelet Logs in OpenSearch

**Symptoms**: Count is 0

**Check 1**: Verify pull script is working
```bash
# Run manually
/usr/local/bin/pull-kubelet-logs.sh

# Check file exists
ls -la /var/log/kubelet/kubelet.log

# View logs
head -20 /var/log/kubelet/kubelet.log
```

**Check 2**: Verify Fluent Bit is reading file
```bash
# Check Fluent Bit logs
journalctl -u fluent-bit-host.service | grep kubelet

# Should see:
# [info] [input:tail:tail.X] inotify_fs_add(): inode=XXXXX watch_fd=1 name=/var/log/kubelet/kubelet.log
```

**Check 3**: Verify cron job
```bash
crontab -l | grep pull-kubelet
# Should see: */5 * * * * /usr/local/bin/pull-kubelet-logs.sh
```

---

### Kubelet Logs Not Updating

**Symptoms**: Same logs, no new data

**Solution**: Manually run pull script
```bash
/usr/local/bin/pull-kubelet-logs.sh
```

**Check minikube is running**:
```bash
su - philip -c "minikube status"
```

**Check kubelet is running in minikube**:
```bash
su - philip -c "minikube ssh 'ps aux | grep kubelet'"
```

---

### Permission Denied Errors

**Symptoms**: Script fails with permission errors

**Solution**: Check minikube ssh access
```bash
# Test minikube ssh
su - philip -c "minikube ssh 'echo test'"

# If fails, restart minikube
su - philip -c "minikube start"
```

---

### Fluent Bit Not Watching File

**Symptoms**: No inotify_fs_add log message

**Solution**: Restart Fluent Bit
```bash
systemctl restart fluent-bit-host.service
```

---

### Journalctl Access Denied

**Symptoms**: Cannot read kubelet journal

**Solution**: Use sudo in minikube ssh
```bash
su - philip -c "minikube ssh 'sudo journalctl -u kubelet -n 10'"
```

---

## Comparison: Kubelet vs Other Logs

| Feature | Kubelet | Container Logs | K8s Events |
|---------|---------|----------------|------------|
| **Source** | Node agent | Container stdout/stderr | K8s API |
| **Content** | Pod management, volumes | Application output | State changes |
| **Location** | systemd journal (minikube) | /var/log/containers/*.log | K8s API |
| **Collection Method** | Pull script + tail | Fluent Bit tail | kubernetes_events input |
| **Update Frequency** | Every 5 min | Real-time | Real-time |

---

## Files and Locations

| File/Path | Purpose |
|-----------|---------|
| `/usr/local/bin/pull-kubelet-logs.sh` | Script to pull logs from minikube |
| `/var/log/kubelet/kubelet.log` | Kubelet logs on host |
| `/etc/fluent-bit/fluent-bit.conf` | Fluent Bit configuration |
| `/var/lib/fluent-bit/flb_kubelet.db` | Fluent Bit database for file position |
| Crontab entry | Scheduled log pulls (every 5 min) |

---

## Maintenance

### Update Pull Frequency

To change from 5 minutes to 1 minute:
```bash
# Edit crontab
crontab -e

# Change:
# */5 * * * * /usr/local/bin/pull-kubelet-logs.sh

# To:
# */1 * * * * /usr/local/bin/pull-kubelet-logs.sh
```

### Change Number of Logs Pulled

Edit `/usr/local/bin/pull-kubelet-logs.sh`:
```bash
# Change from:
journalctl -u kubelet -n 100 --no-pager --output=short

# To:
journalctl -u kubelet -n 500 --no-pager --output=short
```

### View Fluent Bit Statistics

```bash
# Check if file is being watched
journalctl -u fluent-bit-host.service --since "5 minutes ago" | grep kubelet

# Check database file
ls -la /var/lib/fluent-bit/flb_kubelet.db

# Check database content (requires sqlite3)
sqlite3 /var/lib/fluent-bit/flb_kubelet.db "SELECT * FROM main;"
```

---

## Summary

| Component | Status |
|-----------|--------|
| **Kubelet Log Source** | ✅ Active (minikube systemd) |
| **Pull Script** | ✅ Installed and executable |
| **Cron Job** | ✅ Running every 5 minutes |
| **Fluent Bit Config** | ✅ Tailing kubelet log file |
| **OpenSearch Collection** | ✅ 101+ logs collected |
| **Log Type** | `kubelet` |
| **Log Source** | `host` |

---

## Quick Reference Commands

```bash
# Manually pull kubelet logs
/usr/local/bin/pull-kubelet-logs.sh

# View kubelet log file
tail -f /var/log/kubelet/kubelet.log

# Get kubelet logs directly from minikube
su - philip -c "minikube ssh 'sudo journalctl -u kubelet -f'"

# Check Fluent Bit is watching
journalctl -u fluent-bit-host.service | grep -i kubelet

# Count kubelet logs in OpenSearch
curl -s "http://192.168.201.152:9200/unified-logs/_count" \
  -H 'Content-Type: application/json' \
  -d '{"query": {"term": {"log_type.keyword": "kubelet"}}}' | jq '.count'

# Get recent kubelet logs from OpenSearch
curl -s "http://192.168.201.152:9200/unified-logs/_search" \
  -H 'Content-Type: application/json' \
  -d '{
    "query": {"term": {"log_type.keyword": "kubelet"}},
    "size": 10,
    "sort": [{"@timestamp": "desc"}]
  }' | jq '.hits.hits[] | {timestamp: ._source["@timestamp"], log: ._source.log[0:100]}'

# Restart Fluent Bit
systemctl restart fluent-bit-host.service

# View cron job
crontab -l | grep kubelet
```

---

## Related Documentation

- [Fluent Bit Status Report](/root/hynix/FLUENT_BIT_STATUS.md)
- [Log Categorization](/tmp/LOG_CATEGORIZATION.md)
- [Kubernetes Events Guide](/tmp/KUBERNETES_EVENTS_GUIDE.md)
- [Spark Operator README](/tmp/SPARK_OPERATOR_README.md)

---

**Document Version**: 1.0
**Last Updated**: 2026-02-02
**Maintained By**: Data Engineering Team
