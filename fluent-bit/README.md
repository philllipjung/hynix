# Fluent Bit Configuration Files

This directory contains active Fluent Bit configuration files for the Hynix logging infrastructure.

## Files

| File | Purpose | Deployment Target |
|------|---------|-------------------|
| `fluent-bit-k8s.yaml` | Kubernetes DaemonSet configuration | Apply to K8s cluster |
| `fluent-bit-host.conf` | Host systemd service configuration | Copy to `/etc/fluent-bit/` |
| `parsers.conf` | Log parser definitions | Used by both configs |

## Quick Start

### Apply K8s Config
```bash
cp fluent-bit-k8s.yaml /tmp/
chown philip:philip /tmp/fluent-bit-k8s.yaml
su - philip -c "minikube kubectl -- apply -f /tmp/fluent-bit-k8s.yaml"
```

### Apply Host Config
```bash
cp fluent-bit-host.conf /etc/fluent-bit/fluent-bit.conf
systemctl restart fluent-bit-host.service
```

## Documentation

See `/root/hynix/docs/` for detailed guides:
- `/root/hynix/docs/guides/fluent-bit.md` - Complete guide
- `/root/hynix/docs/guides/kubelet.md` - Kubelet log collection
- `/root/hynix/docs/guides/kubernetes-events.md` - K8s events

## Status

✅ All configurations are active and tested
✅ K8s Fluent Bit: Running in `logging` namespace
✅ Host Fluent Bit: Running as `fluent-bit-host.service`

## History

- 2026-02-02: Refactored configs with documentation
- 2026-02-02: Added kubelet log collection
- 2026-02-02: Fixed Spark Operator log collection
