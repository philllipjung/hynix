pipeline {
    agent any

    environment {
        PROVISION_ID = '0002_wfbm'
        GITHUB_REPO = 'https://github.com/philllipjung/hynix.git'
        CONFIG_FILE = 'config/config.json'
        GIT_CREDENTIALS_ID = 'github-creds'
    }

    stages {
        stage('Checkout') {
            steps {
                echo "Checking out repository..."
                git branch: 'master',
                    url: "${GITHUB_REPO}",
                    credentialsId: "${GIT_CREDENTIALS_ID}"
            }
        }

        stage('Checkout') {
            steps {
                echo "Checking out repository..."
                git branch: 'master',
                    url: "${GITHUB_REPO}",
                    credentialsId: "${GIT_CREDENTIALS_ID}"
            }
        }
    }
