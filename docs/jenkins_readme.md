# Jenkins CI/CD Pipeline for Pilot

## Overview

Automated Jenkins pipeline that updates `build_number.number` in `config/config.json` for a specific `provision_id` and commits changes to the repository.

**Repository**: https://github.com/philllipjung/pilot.git
**Job Name**: `pilot-pipeline`
**Target Provision ID**: `0002_wfbm`

---

## Jenkins Access

| Field | Value |
|-------|-------|
| URL | http://localhost:8086 |
| Username | `admin` |
| Password | `ba6e43ad1ae34d5d860da226dca32aa8` |
| Job URL | http://localhost:8086/job/pilot-pipeline/ |

---

## What the Pipeline Does

Each Jenkins build:

1. **Clones** the pilot repository from GitHub
2. **Updates** `config_specs.build_number.number` for `provision_id: "0002_wfbm"`
3. **Commits** the changes with message: `Update build_number to <BUILD_NUMBER>`
4. **Pushes** to the `master` branch

### Example

```
Build #1 → build_number.number = "1"
Build #2 → build_number.number = "2"
Build #3 → build_number.number = "3"
```

---

## Quick Start

### Change Jenkins Port (Optional)

**Current Port Conflict**: Jenkins uses 8085, API server uses 8080

**Option 1: Change Jenkins Port**
```bash
# Stop Jenkins
docker stop jenkins

# Start Jenkins on new port 8086
docker run -d \
  --name jenkins \
  --restart unless-stopped \
  -p 8086:8080 \
  -p 50000:50000 \
  -v jenkins_home:/var/jenkins_home \
  jenkins:jenkins.2.426.3-jdk11

# Or modify docker-compose.yml if using compose
```

**Option 2: Use Different Hosts**
- Keep Jenkins on 8085 (localhost:8085)
- API server on 8080 (localhost:8080)
- Proxy server on 8082 (localhost:8082)

All three services can run together without conflicts.

---

### Trigger a Build

**Option 1: Web UI**
1. Open http://localhost:8086/job/pilot-pipeline/
2. Click "Build Now"

**Option 2: CLI**
```bash
docker exec jenkins java -jar /tmp/jenkins-cli.jar -s http://localhost:8080 \
  -auth admin:ba6e43ad1ae34d5d860da226dca32aa8 build pilot-pipeline
```

**Option 3: API**
```bash
curl -X POST http://localhost:8086/job/pilot-pipeline/build \
  --user "admin:ba6e43ad1ae34d5d860da226dca32aa8"
```

### Check Build Status

```bash
# Latest build
curl --user "admin:ba6e43ad1ae34d5d860da226dca32aa8" \
  http://localhost:8086/job/pilot-pipeline/lastBuild/api/json

# Console output
curl --user "admin:ba6e43ad1ae34d5d860da226dca32aa8" \
  http://localhost:8086/job/pilot-pipeline/lastBuild/consoleText
```

---

## Configuration

### Target Configuration

The pipeline updates `config/config.json`:

```json
{
  "config_specs": [
    {
      "provision_id": "0002_wfbm",
      "enabled": "true",
      "build_number": {
        "number": "<JENKINS_BUILD_NUMBER>"
      }
    }
  ]
}
```

### Parameters

| Parameter | Default | Description |
|-----------|---------|-------------|
| `PROVISION_ID` | `0002_wfbm` | Provision ID to update in config |

---

## Job Configuration

The job is a **FreeStyle Project** with a Shell builder:

```bash
# Clone repository using Jenkins credentials
git clone https://github.com/philllipjung/hynix.git

# Update build_number using sed
sed -i "/\"provision_id\": \"$PROVISION_ID\"/,+20 { /\"number\":/ s/\"[0-9.]*\"/\"$BUILD_NUMBER\"/ }" config/config.json

# Commit and push
git config user.email "jenkins@hynix.local"
git config user.name "Jenkins Agent"
git add config/config.json
git commit -m "Update build_number to $BUILD_NUMBER"
git push origin master
```

---

## Troubleshooting

### Build Fails - Python Not Found

**Solution**: The job uses `sed` instead of Python, so this shouldn't occur. If you see this error, verify the job configuration.

### Git Push Fails - Authentication Error

**Cause**: GitHub token is invalid or expired

**Solution**:
1. Update the token in the job configuration
2. Job config location: `/var/jenkins_home/jobs/pilot-pipeline/config.xml`

### Config File Not Found

**Cause**: Repository structure changed

**Solution**:
```bash
# Verify config exists in repo
git clone https://github.com/philllipjung/pilot.git
ls pilot/config/config.json
```

### Build Number Not Updated

**Cause**: `sed` pattern doesn't match the provision_id

**Solution**:
1. Check the provision_id matches exactly
2. Verify the config.json format
3. Check build console output for errors

---

## Jenkins Management

### Start/Stop Jenkins

```bash
# Stop
docker stop jenkins

# Start
docker start jenkins

# Restart
docker restart jenkins

# View logs
docker logs -f jenkins
```

### Backup Job Configuration

```bash
# Export job config
docker exec jenkins cat /var/jenkins_home/jobs/pilot-pipeline/config.xml > /tmp/pilot-pipeline-config.xml

# Import job config
docker cp /tmp/pilot-pipeline-config.xml jenkins:/var/jenkins_home/jobs/pilot-pipeline/config.xml
docker restart jenkins
```

---

## File Locations

| File | Location |
|------|----------|
| Job Config | `/var/jenkins_home/jobs/pilot-pipeline/config.xml` |
| Jenkins Home | `/var/jenkins_home/` |
| Build Workspace | `/var/jenkins_home/workspace/pilot-pipeline/` |
| Jenkins CLI | `/tmp/jenkins-cli.jar` (in container) |

---

## API Reference

### Job API
```bash
# Job details
curl --user "admin:PASSWORD" http://localhost:8086/job/pilot-pipeline/api/json

# Build history
curl --user "admin:PASSWORD" http://localhost:8086/job/pilot-pipeline/api/json?tree=builds[number,result,timestamp]

# Job configuration
curl --user "admin:PASSWORD" http://localhost:8086/job/pilot-pipeline/config.xml
```

### Build API
```bash
# Latest build info
curl --user "admin:PASSWORD" http://localhost:8086/job/pilot-pipeline/lastBuild/api/json

# Specific build
curl --user "admin:PASSWORD" http://localhost:8086/job/pilot-pipeline/3/api/json

# Console text
curl --user "admin:PASSWORD" http://localhost:8086/job/pilot-pipeline/3/consoleText
```

### Queue API
```bash
# Check queue
curl --user "admin:PASSWORD" http://localhost:8086/queue/api/json
```

---

## GitHub Repository

**Repository**: https://github.com/philllipjung/pilot.git
**Branch**: `master`
**Config File**: `config/config.json`

### Config Structure

```json
{
  "config_specs": [
    {
      "provision_id": "0002_wfbm",
      "enabled": "true",
      "resource_calculation": {
        "minio": "1234/5678",
        "threshold": 10000000,
        "min_queue": "min",
        "max_queue": "max"
      },
      "gang_scheduling": {
        "cpu": "5",
        "memory": "10",
        "executor": "1"
      },
      "build_number": {
        "number": "3"
      }
    }
  ]
}
```

---

## Recent Build History

| Build | Result | Commit | Message |
|-------|--------|--------|---------|
| #3 | SUCCESS | 5abbe27 | Update build_number to 3 |
| #2 | SUCCESS | 48e4d8b | Update build_number to 2 |
| #1 | FAILURE | - | Config file not found (initial test) |

---

## Security Notes

⚠️ **Important**: The GitHub token is embedded in the job configuration. For production:

1. Use Jenkins Credentials store
2. Reference credentials by ID in job configuration
3. Rotate tokens regularly

---

## Support

For issues or questions:

1. Check Jenkins logs: `docker logs jenkins`
2. Check build console output in Jenkins UI
3. Verify repository access and token validity
