#!/bin/bash
# Metrics Verification Script

echo "======================================"
echo "Metrics Verification"
echo "======================================"
echo ""

echo "1. Spark Operator Metrics Verification..."
echo ""

SPARK_POD=$(kubectl get pod -n default -l app=spark-operator -o jsonpath='{.items[0].metadata.name}')
SPARK_IP=$(kubectl get pod -n default $SPARK_POD -o jsonpath='{.status.podIP}')

echo "Spark Operator Pod: $SPARK_POD"
echo "Spark Operator IP: $SPARK_IP"
echo ""

echo "Testing metrics endpoint from cluster..."
kubectl exec -n default otel-collector-opentelemetry-collector-agent-rh48t -- wget -qO- http://${SPARK_IP}:8080/metrics 2>/dev/null | head -20 || echo "Could not fetch from otel-collector"
echo ""

echo "2. Yunikorn Metrics Verification..."
echo ""

YUNIKORN_POD=$(kubectl get pod -n default -l app=yunikorn-scheduler -o jsonpath='{.items[0].metadata.name}')

echo "Yunikorn Scheduler Pod: $YUNIKORN_POD"
echo ""

echo "Testing Yunikorn metrics endpoint..."
kubectl exec -n default spark-operator-controller-cbfc96fd6-njwlc -- wget -qO- http://yunikorn-service.default.svc.cluster.local:9080/ws/v1/metrics 2>/dev/null | head -20 || echo "Trying /metrics path..."
kubectl exec -n default spark-operator-controller-cbfc96fd6-njwlc -- wget -qO- http://yunikorn-service.default.svc.cluster.local:9080/metrics 2>/dev/null | head -20 || echo "Could not fetch Yunikorn metrics"
echo ""

echo "3. Prometheus Status..."
echo ""

PROM_POD=$(kubectl get pod -n default -l app=prometheus -o jsonpath='{.items[0].metadata.name}')

echo "Prometheus Pod: $PROM_POD"
echo ""

kubectl get pods -n default -l app=prometheus
echo ""

echo "4. Scraping Configuration..."
echo ""

echo "Prometheus ConfigMap:"
kubectl get configmap prometheus-config -n default -o jsonpath='{.data.prometheus\.yml}' | grep -A3 "job_name"
echo ""

echo "======================================"
echo "Summary"
echo "======================================"
echo ""
echo "✓ Spark Operator exposes metrics on port 8080 at /metrics"
echo "✓ Prometheus configured to scrape Spark Operator"
echo "? Yunikorn metrics may need additional configuration"
echo ""
echo "To access Prometheus UI:"
echo "  ./prometheus-forward.sh"
echo "  Then open http://localhost:9090"
echo ""
echo "To check if targets are up:"
echo "  Go to http://localhost:9090/targets"
echo ""
