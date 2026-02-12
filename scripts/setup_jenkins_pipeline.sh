#!/bin/bash

set -e

echo "=========================================="
echo "Jenkins Pipeline Setup"
echo "=========================================="
echo ""

# Check if Jenkins is running
if ! docker ps | grep -q jenkins; then
    echo "ERROR: Jenkins is not running!"
    echo "Run: bash /root/hynix/scripts/setup_jenkins.sh"
    exit 1
fi

JENKINS_URL="http://localhost:8085"
JENKINS_USER="admin"
JENKINS_PASS="admin123"

echo "Jenkins URL: ${JENKINS_URL}"
echo "Waiting for Jenkins to be ready..."
sleep 10

# Get initial admin password
ADMIN_PASS=$(docker exec jenkins cat /var/jenkins_home/secrets/initialAdminPassword 2>/dev/null || echo "ba6e43ad1ae34d5d860da226dca32aa8")

echo "Admin Password: ${ADMIN_PASS}"
echo ""

# Install Jenkins CLI (jenkins-plugin-manager)
echo "Installing Jenkins CLI..."
docker exec jenkins jenkins-plugin-cli --version 2>/dev/null || echo "CLI not installed, will install plugins if needed"
echo ""

# Create a job using Jenkins REST API
echo ""
echo "Creating pipeline job via REST API..."

# Create pipeline job
cat > /tmp/create_job.json << 'EOF'
{
  "name": "pliot-pipeline",
  "description": "CI/CD pipeline for ploitory application - updates config.json build_number",
  "_class": "org.jenkinsci.plugins.workflow.job.WorkflowJob",
  "definition": {
    "cps": {
      "script": "@('Jenkinsfile')"
    },
    "sandbox": true
  },
  "properties": [
    {
      "name": "provision_id",
      "value": "${PROVISION_ID}"
    }
  ]
}
EOF

# Create the job
echo "Creating pipeline job: ploitory-pipeline"
curl -X POST "${JENKINS_URL}/createItem?name=pliot-pipeline" \
  --user "${JENKINS_USER}:${ADMIN_PASS}" \
  -H "Content-Type: application/json" \
  -d @/tmp/create_job.json \
  -H "Jenkins-Crumb: .crumb" 2>/dev/null || echo "Job may already exist"

echo ""
echo "=========================================="
echo "Setup Instructions"
echo "=========================================="
echo ""
echo "1. Open Jenkins: http://localhost:8085"
echo "   Username: ${JENKINS_USER}"
echo "   Password: ${ADMIN_PASS}"
echo ""
echo "2. Configure GitHub credentials:"
echo "   - Go to: Manage Jenkins → Credentials → Global credentials"
echo "   - Click: Add Credentials → Kind: 'Username with password'"
echo "   - Username: <your GitHub username>"
echo "   - Password: <GitHub personal access token or password>"
echo "   - ID: github-creds"
echo ""
echo "3. Configure Git tool:"
echo "   - Go to: Manage Jenkins → Global Tool Configuration"
echo "   - Git → Add Git"
echo "   - Name: git"
echo "   - Path: /usr/bin/git"
echo ""
echo "4. Build the pipeline:"
echo "   - Open: http://localhost:8085/job/pliot-pipeline"
echo "   - Click 'Build Now'"
echo "   - Pipeline will:"
echo "     a. Pull github.com/philllipjung/pliot"
echo "     b. Find config_specs.provision_id: ${PROVISION_ID}"
echo "     c. Update build_number.number to Jenkins build number"
echo "     d. Commit changes to git"
echo "     e. Push to repository"
echo ""
echo "5. Monitor logs:"
echo "   - Job logs: http://localhost:8085/job/pliot-pipeline/"
echo "   - Console output: http://localhost:8085/computer/"
echo ""
echo "=========================================="
echo "Pipeline Configuration"
echo "=========================================="
echo ""
echo "Job Name: ploitory-pipeline"
echo "Script Path: @('Jenkinsfile') (from root of repo)"
echo "Variable PROVISION_ID: ${PROVISION_ID}"
echo "GitHub Repository: github.com/philllipjung/pliot"
echo "Config File: config/config.json"
echo ""
