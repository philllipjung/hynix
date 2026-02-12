#!/bin/bash
# OpenTelemetry Collector Deployment Script
# This script deploys OTEL Collector with OpenSearch exporter

set -e

NAMESPACE=${NAMESPACE:-default}
OPENSEARCH_ENDPOINT=${OPENSEARCH_ENDPOINT:-http://opensearch.opensearch.svc.cluster.local:9200}

echo "Deploying OpenTelemetry Collector..."
echo "Namespace: $NAMESPACE"
echo "OpenSearch Endpoint: $OPENSEARCH_ENDPOINT"

# Step 1: Apply RBAC
echo ""
echo "Step 1: Applying RBAC configuration..."
kubectl apply -f rbac.yaml

# Step 2: Apply ConfigMap
echo ""
echo "Step 2: Applying ConfigMap..."
kubectl apply -f configmap.yaml

# Step 3: Install OTEL Collector via Helm
echo ""
echo "Step 3: Installing OTEL Collector via Helm..."
helm repo add open-telemetry https://open-telemetry.github.io/opentelemetry-helm-charts 2>/dev/null || true
helm repo update

helm upgrade --install otel-collector open-telemetry/opentelemetry-collector \
  --namespace $NAMESPACE \
  --set mode=daemonset \
  --set image.repository="otel/opentelemetry-collector-contrib" \
  --set image.tag="0.122.0" \
  --set configMap.name=otel-collector-opentelemetry-collector-agent \
  --set configMap.key=relay \
  --set serviceAccount.name=otel-collector-opentelemetry-collector \
  --set serviceAccount.create=false

# Step 4: Patch DaemonSet for volume mounts
echo ""
echo "Step 4: Patching DaemonSet for volume mounts..."
kubectl patch daemonset otel-collector-opentelemetry-collector-agent \
  -n $NAMESPACE \
  --type='json' \
  -p="$(cat volume-patch.yaml)"

# Step 5: Wait for pods to be ready
echo ""
echo "Step 5: Waiting for OTEL Collector pods to be ready..."
kubectl wait --for=condition=ready pod \
  -l app.kubernetes.io/name=opentelemetry-collector \
  -n $NAMESPACE \
  --timeout=60s

echo ""
echo "✓ OpenTelemetry Collector deployed successfully!"
echo ""
echo "To verify the deployment:"
echo "  kubectl get pods -n $NAMESPACE -l app.kubernetes.io/name=opentelemetry-collector"
echo ""
echo "To view logs:"
echo "  kubectl logs -n $NAMESPACE -l app.kubernetes.io/name=opentelemetry-collector --tail=50"
echo ""
echo "To check metrics:"
echo "  kubectl port-forward -n $NAMESPACE svc/otel-collector-opentelemetry-collector-agent 8888:8888"
echo "  curl http://localhost:8888/metrics"
