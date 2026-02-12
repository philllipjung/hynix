#!/bin/bash
set -e

echo "Removing old Jenkins container..."
docker rm -f jenkins 2>/dev/null || true

echo "Starting Jenkins on port 8085..."
docker run -d --name jenkins \
  -p 8085:8080 \
  -p 50000:50000 \
  -v jenkins_home:/var/jenkins_home \
  -e JENKINS_ADMIN_ID=admin \
  -e JENKINS_ADMIN_PASSWORD=admin123 \
  jenkins/jenkins:2.426.3-jdk11

echo "Waiting for Jenkins to start (this may take a minute)..."
sleep 30

echo ""
echo "Checking Jenkins status..."
docker ps -a | grep jenkins

echo ""
echo "Getting initial admin password..."
sleep 5
docker exec jenkins cat /var/jenkins_home/secrets/initialAdminPassword 2>/dev/null || echo "Password: admin123 (set via env var)"

echo ""
echo "Jenkins is running at: http://localhost:8085"
echo "Username: admin"
echo "Password: admin123"
echo ""
echo "Waiting for initialization..."
sleep 30

echo ""
echo "Checking logs..."
docker logs jenkins 2>&1 | grep -i "Jenkins is fully up" || echo "Check logs: docker logs jenkins"
