# Jenkins Pipeline Complete Guide

## 목차

1. [개요](#1-개요)
2. [아키텍처](#2-아키텍처)
3. [설치](#3-설치)
4. [설정](#4-설정)
5. [파이프라인](#5-파이프라인)
6. [빌드 번호 관리](#6-빌드-번호-관리)
7. [실행 및 검증](#7-실행-및-검증)
8. [문제 해결](#8-문제-해결)

---

## 1. 개요

### 시스템 구성

본 시스템은 **Jenkins Pipeline**을 사용하여 빌드 번호를 자동으로 관리합니다.

| 구분 | 설명 |
|------|------|
| **Jenkins** | Docker 컨테이너로 실행 (Port: 8081) |
| **Pipeline** | Jenkinsfile 기반 declarative pipeline |
| **저장소** | https://github.com/philllipjung/hynix |
| **대상 프로비저닝** | 0002_wfbm |

### 빌드 번호 관리

- **파일**: `config/config.json`
- **경로**: `.config_specs[].build_number.number`
- **업데이트**: Jenkins 빌드 시 자동 증가

---

## 2. 아키텍처

```
┌─────────────────────────────────────────────────────────────┐
│                       Jenkins                                │
│  ┌──────────────────────────────────────────────────────┐   │
│  │  Pipeline: hynix-pipeline                            │   │
│  │                                                       │   │
│  │  1. Checkout (git clone)                             │   │
│  │  2. Read Config (config.json)                       │   │
│  │  3. Update Build Number (BUILD_NUMBER)               │   │
│  │  4. Commit (git commit)                              │   │
│  │  5. Push (git push)                                  │   │
│  └──────────────────────────────────────────────────────┘   │
│                           │                                  │
└───────────────────────────┼───────────────────────────┘     │
                            │                                   │
┌───────────────────────────┴─────────────────────────────┐   │
│              GitHub Repository                              │
│  https://github.com/philllipjung/hynix                     │
│                                                           │
│  - Jenkinsfile (Pipeline 정의)                             │
│  - config/config.json (빌드 번호 저장)                     │
│                                                           │
└───────────────────────────────────────────────────────┘     │
```

---

## 3. 설치

### 3.1 Jenkins Docker 실행

```bash
# Jenkins 볼륨 생성
docker volume create jenkins_home

# Jenkins 실행
docker run -d \
  --name jenkins \
  --restart unless-stopped \
  -p 8081:8080 \
  -p 50000:50000 \
  -v jenkins_home:/var/jenkins_home \
  jenkins/jenkins:lts

# 초기 관리자 비밀번호 확인
docker exec jenkins cat /var/jenkins_home/secrets/initialAdminPassword
```

**접속 정보**:
- URL: http://192.168.201.152:8081
- 초기 비밀번호: `14059ab732084b90a9fc1a2dcb3f1de6`

### 3.2 필수 플러그인

Jenkins 초기 설정 시 다음 플러그인 설치:
- **Pipeline** (필수)
- **Git** (필수)
- **GitHub Integration**

---

## 4. 설정

### 4.1 GitHub 자격증명 추가

**경로**: Jenkins → Manage Jenkins → Credentials → Global credentials → Add Credentials

**설정**:
- Kind: **Username with password**
- Username: `philllipjung`
- Password: `<GITHUB_TOKEN>` (저장소 관리자에게 문의)
- ID: `github-service-comm`
- Description: `GitHub Token`

### 4.2 Jenkins Job 생성

**경로**: Jenkins → New Item

**설정**:
1. Job name: `hynix-pipeline`
2. Type: **Pipeline**
3. Pipeline → Definition: **Pipeline script from SCM**
4. SCM: **Git**
5. Repository URL: `https://github.com/philllipjung/hynix.git`
6. Script Path: `Jenkinsfile`

---

## 5. 파이프라인

### 5.1 Jenkinsfile

**위치**: `/root/hynix/Jenkinsfile`

```groovy
pipeline {
    agent any

    environment {
        REPO_URL = 'https://github.com/philllipjung/hynix.git'
    }

    stages {
        stage('Checkout') {
            steps {
                git url: "${REPO_URL}", branch: 'main'
            }
        }

        stage('Read Config') {
            steps {
                script {
                    // config.json 읽기
                    def configJson = readJSON file: 'config/config.json'

                    // provision_id: 0002_wfbm 찾기
                    def spec = configJson.config_specs.find {
                        it.provision_id == '0002_wfbm'
                    }

                    if (spec) {
                        def currentBuildNum = spec.build_number?.number ?: '0'
                        echo "Current build number: ${currentBuildNum}"
                        echo "Jenkins build number: ${BUILD_NUMBER}"

                        // 빌드 번호 업데이트
                        spec.build_number = [
                            number: "${BUILD_NUMBER}"
                        ]

                        // 파일에 쓰기
                        writeJSON file: 'config/config.json', json: configJson
                        echo "✅ Updated build number to: ${BUILD_NUMBER}"
                    } else {
                        error "provision_id 0002_wfbm not found"
                    }
                }
            }
        }

        stage('Commit Changes') {
            steps {
                withCredentials([usernamePassword(
                    credentialsId: 'github-service-comm',
                    usernameVariable: 'GITHUB_USER',
                    passwordVariable: 'GITHUB_TOKEN'
                )]) {
                    sh '''
                        git config user.name "Jenkins CI"
                        git config user.email "jenkins@ci.local"
                        git add config/config.json
                        git commit -m "Build ${BUILD_NUMBER}: Update build number for 0002_wfbm" || echo "No changes"
                        git push https://${GITHUB_USER}:${GITHUB_TOKEN}@github.com/philllipjung/hynix.git HEAD:main
                        echo "✅ Push completed!"
                    '''
                }
            }
        }
    }

    post {
        success {
            echo "✅ Build ${BUILD_NUMBER} - SUCCESS"
            echo "config/config.json updated successfully"
        }
        failure {
            echo "❌ Build ${BUILD_NUMBER} - FAILED"
        }
    }
}
```

---

## 6. 빌드 번호 관리

### 6.1 빌드 번호 흐름

```
Jenkins Build #1 → BUILD_NUMBER=1 → config.json → "1"
Jenkins Build #2 → BUILD_NUMBER=2 → config.json → "2"
Jenkins Build #3 → BUILD_NUMBER=3 → config.json → "3"
...
Jenkins Build #10 → BUILD_NUMBER=10 → config.json → "10"
```

### 6.2 config.json 구조

```json
{
  "config_specs": [
    {
      "provision_id": "0002_wfbm",
      "enabled": "true",
      "resource_calculation": {
        "minio": "/root/hynix/kubernetes.zip",
        "threshold": 10,
        "min_queue": "min",
        "max_queue": "max"
      },
      "gang_scheduling": {
        "cpu": "5",
        "memory": "10",
        "executor": "1"
      },
      "build_number": {
        "number": "11"
      }
    }
  ]
}
```

### 6.3 빌드 번호 확인

```bash
# Jenkins UI
http://192.168.201.152:8081/job/hynix-pipeline/

# Git 저장소
cd /root/hynix
cat config/config.json | jq '.config_specs[] | select(.provision_id == "0002_wfbm")'

# GitHub 원격
curl -s "https://raw.githubusercontent.com/philllipjung/hynix/main/config/config.json" | \
  jq '.config_specs[] | select(.provision_id == "0002_wfbm")'
```

---

## 7. 실행 및 검증

### 7.1 수동 빌드 실행

```bash
# Jenkins UI에서
1. http://192.168.201.152:8081/job/hynix-pipeline/
2. "Build Now" 클릭
3. 빌드 번호 자동 증가
```

### 7.2 E2E 테스트

```bash
#!/bin/bash

# E2E 테스트 스크립트
JENKINS_BUILD_NUM=20

# Jenkins 워크스페이스 시뮬레이션
mkdir -p /tmp/jenkins-test
cd /tmp/jenkins-test
rm -rf hynix

# 1. Clone
git clone https://github.com/philllipjung/hynix.git
cd hynix

# 2. Update build number
cat config/config.json | \
  jq "(.config_specs[] | select(.provision_id == \"0002_wfbm\") | .build_number.number) |= \"$JENKINS_BUILD_NUM\"" \
  > config/config.json.tmp
mv config/config.json.tmp config/config.json

# 3. Commit
git config user.name "Jenkins CI"
git config user.email "jenkins@ci.local"
git add config/config.json
git commit -m "Build $JENKINS_BUILD_NUM: Update build number for 0002_wfbm"

# 4. Push
TOKEN="<GITHUB_TOKEN>"
git push https://philllipjung:$TOKEN@github.com/philllipjung/hynix.git main

# 5. Verify
curl -s "https://raw.githubusercontent.com/philllipjung/hynix/main/config/config.json" | \
  jq '.config_specs[] | select(.provision_id == "0002_wfbm") | .build_number'
```

### 7.3 빌드 결과 확인

```bash
# Jenkins 콘솔 출력
http://192.168.201.152:8081/job/hynix-pipeline/lastBuild/consoleText

# 빌드 상태
curl -s -u admin:14059ab732084b90a9fc1a2dcb3f1de6 \
  "http://192.168.201.152:8081/job/hynix-pipeline/lastBuild/api/json" | \
  jq '{id, result, duration}'

# Git 커밋 로그
cd /root/hynix
git log --oneline -5
```

---

## 8. 문제 해결

### 8.1 GitHub Push 실패

**에러**:
```
remote: Permission denied
fatal: unable to access
```

**해결**:
1. GitHub 자격증명 확인: `github-service-comm`
2. 토큰 유효성 확인
3. 저장소 쓰기 권한 확인

### 8.2 config.json 파싱 오류

**에러**:
```
provision_id 0002_wfbm not found
```

**해결**:
```bash
# config.json 구조 검증
cat config/config.json | jq '.config_specs[] | .provision_id'

# provision_id 확인
cat config/config.json | jq '.config_specs[] | select(.provision_id == "0002_wfbm")'
```

### 8.3 Jenkins Job 로드 실패

**에러**:
```
Failed to load job hynix-pipeline
```

**해결**:
```bash
# Job 재생성
docker exec jenkins mkdir -p /var/jenkins_home/jobs/hynix-pipeline
# config.xml 생성 후 Jenkins 재시작
docker restart jenkins
```

### 8.4 CSRF 오류

**에러**:
```
HTTP ERROR 403 No valid crumb was included in the request
```

**해결**:
- Jenkins Script Console 사용
- 또는 파일 시스템 직접 수정

---

## 9. 파일 위치

### Jenkins 관련

| 파일/경로 | 설명 |
|-----------|------|
| **Jenkinsfile** | `/root/hynix/Jenkinsfile` |
| **Job Config** | `/var/jenkins_home/jobs/hynix-pipeline/config.xml` |
| **Credentials** | `/var/jenkins_home/credentials.xml` |
| **로그** | Docker logs: `docker logs jenkins` |

### 문서

| 파일 | 설명 |
|------|------|
| **README** | `/root/hynix/JENKINS_README.md` (이 파일) |
| **GitHub Actions README** | `/root/hynix/FLUENT_BIT_README.md` |
| **Config 파일** | `/root/hynix/config/config.json` |

---

## 10. 참고

### Jenkins

- **URL**: http://192.168.201.152:8081
- **Job**: http://192.168.201.152:8081/job/hynix-pipeline/
- **CLI**: `docker exec jenkins java -jar /var/jenkins_home/war/WEB-INF/jenkins-cli.jar`

### GitHub

- **Repository**: https://github.com/philllipjung/hynix
- **Jenkinsfile**: https://github.com/philllipjung/hynix/blob/main/Jenkinsfile
- **Config**: https://github.com/philllipjung/hynix/blob/main/config/config.json

### 관련 문서

- `/root/hynix/README.md`
- `/root/hynix/FLUENT_BIT_README.md`

---

**버전**: Jenkins 2.541.1 (LTS Docker)
**마지막 업데이트**: 2026-01-30
