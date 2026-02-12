#!/bin/bash
# Fluent Bit wrapper script with port-forward

# Kill any existing port-forwards
pkill -f "port-forward.*opensearch" 2>/dev/null
sleep 1

# Start port-forward in background
kubectl port-forward -n opensearch svc/opensearch 9200:9200 > /dev/null 2>&1 &
PF_PID=$!
sleep 3

# Check if port-forward is running
if ! ps -p $PF_PID > /dev/null; then
    echo "ERROR: Port-forward failed to start"
    exit 1
fi

echo "Port-forward started (PID: $PF_PID)"
echo "Starting Fluent Bit..."

# Start fluent-bit
exec /usr/local/bin/fluent-bit -c /etc/fluent-bit/fluent-bit.conf
