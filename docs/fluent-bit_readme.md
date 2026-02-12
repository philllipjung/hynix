# Fluent Bit - Hynix Log Collector

Fluent Bit collects logs from Hynix applications and forwards them to OpenSearch for indexing and search.

## Overview

- **Fluent Bit Version**: v2.2.0
- **Log Sources**:
  - `/root/hynix/hynix.log` (tag: `hynix`)
  - `/root/hynix/proxy.log` (tag: `proxy`)
- **Destination**: OpenSearch index `hynix_logs`
- **Format**: JSON

## Architecture

```
┌─────────────────┐     ┌──────────────┐     ┌─────────────┐
│  hynix.log      │     │              │     │             │
│  proxy.log      │────▶│ Fluent Bit   │────▶│  OpenSearch │
│                 │     │              │     │ (hynix_logs)│
└─────────────────┘     └──────────────┘     └─────────────┘
```

## Configuration Files

### Main Config: `/etc/fluent-bit/fluent-bit.conf`

```ini
[SERVICE]
    Flush         5
    Daemon        off
    Log_Level     info
    Parsers_File  parsers.conf

[INPUT]
    Name              tail
    Path              /root/hynix/hynix.log
    Tag               hynix
    Parser            json
    Read_from_Head    true

[INPUT]
    Name              tail
    Path              /root/hynix/proxy.log
    Tag               proxy
    Parser            json
    Read_from_Head    true

[OUTPUT]
    Name            opensearch
    Match           *
    Host            127.0.0.1
    Port            9200
    Index           hynix_logs
    Suppress_Type_Name on
```

### Parsers Config: `/etc/fluent-bit/parsers.conf`

```ini
[PARSER]
    Name        json
    Format      json
```

## Running Fluent Bit

### Manual Start with Port-Forward

```bash
# Start OpenSearch port-forward
kubectl port-forward -n opensearch svc/opensearch 9200:9200 > /tmp/os-pf.log 2>&1 &

# Run Fluent Bit
/usr/local/bin/fluent-bit -c /etc/fluent-bit/fluent-bit.conf
```

### Using the Wrapper Script

```bash
/root/hynix/scripts/start-fluent-bit.sh
```

This script automatically:
1. Kills any existing port-forwards
2. Starts kubectl port-forward for OpenSearch
3. Starts Fluent Bit

## Querying Logs in OpenSearch

### Query All Logs

```bash
curl -s "http://127.0.0.1:9200/hynix_logs/_search?pretty&size=10"
```

### Query by Tag

```bash
# Hynix logs only
curl -s "http://127.0.0.1:9200/hynix_logs/_search?pretty" \
  -H "Content-Type: application/json" \
  -d '{
    "query": {"term": {"tag": "hynix"}},
    "size": 10
  }'

# Proxy logs only
curl -s "http://127.0.0.1:9200/hynix_logs/_search?pretty" \
  -H "Content-Type: application/json" \
  -d '{
    "query": {"term": {"tag": "proxy"}},
    "size": 10
  }'
```

### Count Documents

```bash
curl -s "http://127.0.0.1:9200/hynix_logs/_count?pretty"
```

### Time Range Query

```bash
curl -s "http://127.0.0.1:9200/hynix_logs/_search?pretty" \
  -H "Content-Type: application/json" \
  -d '{
    "size": 10,
    "sort": [{"@timestamp": {"order": "desc"}}],
    "query": {
      "range": {
        "@timestamp": {
          "gte": "now-1h"
        }
      }
    }
  }'
```

## OpenSearch Dashboard

Access the OpenSearch Dashboard via port-forward:

```bash
kubectl port-forward -n opensearch svc/opensearch-dashboards 5601:5601
```

Then open: http://localhost:5601

## Troubleshooting

### Check Fluent Bit Status

```bash
# View real-time logs
/usr/local/bin/fluent-bit -c /etc/fluent-bit/fluent-bit.conf

# Check if port-forward is running
ps aux | grep port-forward
```

### Test Configuration

```bash
/usr/local/bin/fluent-bit -c /etc/fluent-bit/fluent-bit.conf --dry-run
```

### Verify OpenSearch Connection

```bash
curl http://127.0.0.1:9200/_cluster/health?pretty
```

### View Received Logs

```bash
curl -s "http://127.0.0.1:9200/hynix_logs/_search?pretty&size=5"
```

## Kubernetes Services

| Service | Namespace | Type | Port |
|---------|-----------|------|------|
| opensearch | opensearch | ClusterIP | 9200, 9600 |
| opensearch-dashboards | opensearch | ClusterIP | 5601 |

## Files Reference

| Path | Purpose |
|------|---------|
| `/etc/fluent-bit/fluent-bit.conf` | Main Fluent Bit configuration |
| `/etc/fluent-bit/parsers.conf` | JSON parser definition |
| `/root/hynix/scripts/start-fluent-bit.sh` | Wrapper script with port-forward |
| `/root/hynix/hynix.log` | Hynix application log source |
| `/root/hynix/proxy.log` | Proxy log source |
