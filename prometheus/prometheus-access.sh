#!/bin/bash
# Prometheus Access and Verification Script

echo "======================================"
echo "Prometheus Setup Verification"
echo "======================================"
echo ""

echo "1. Checking Prometheus Pod..."
kubectl get pods -n default -l app=prometheus
echo ""

echo "2. Checking Prometheus Service..."
kubectl get svc prometheus -n default
echo ""

echo "3. Checking RBAC..."
kubectl get serviceaccount prometheus -n default
echo ""

echo "4. Checking ConfigMap..."
kubectl get configmap prometheus-config -n default
echo ""

echo "======================================"
echo "Access Prometheus UI"
echo "======================================"
echo ""
echo "To access the Prometheus web UI, run:"
echo ""
echo "  ./prometheus-forward.sh"
echo ""
echo "Then open in your browser:"
echo "  http://localhost:9090"
echo ""
echo "======================================"
echo "Quick Start Queries"
echo "======================================"
echo ""
echo "Yunikorn Metrics:"
echo "  - yunikorn_scheduler_total_applications"
echo "  - yunikorn_scheduler_queue_allocated_memory"
echo "  - yunikorn_scheduler_queue_allocated_vcore"
echo ""
echo "Spark Operator Metrics:"
echo "  - spark_operator_spark_applications_running"
echo "  - spark_operator_applications_total"
echo "  - spark_operator_spark_applications_completed"
echo ""
echo "======================================"
echo "For more information, see:"
echo "  PROMETHEUS.md"
echo "======================================"
