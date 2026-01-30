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
                    // Read config.json
                    def configJson = readJSON file: 'config/config.json'
                    
                    // Find 0002_wfbm
                    def spec = configJson.config_specs.find { it.provision_id == '0002_wfbm' }
                    
                    if (spec) {
                        def currentBuildNum = spec.build_number?.number ?: '0'
                        echo "=========================================="
                        echo "Current build number: ${currentBuildNum}"
                        echo "Jenkins build number: ${BUILD_NUMBER}"
                        echo "=========================================="
                        
                        // Update with Jenkins BUILD_NUMBER
                        spec.build_number = [
                            number: "${BUILD_NUMBER}"
                        ]
                        
                        // Write back
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
                        echo "Committing changes..."
                        git config user.name "Jenkins CI"
                        git config user.email "jenkins@ci.local"
                        git add config/config.json
                        git commit -m "Build ${BUILD_NUMBER}: Update build number for 0002_wfbm" || echo "No changes to commit"
                        echo "Pushing to GitHub..."
                        git push https://${GITHUB_USER}:${GITHUB_TOKEN}@github.com/philllipjung/hynix.git HEAD:main
                        echo "✅ Push completed!"
                    '''
                }
            }
        }
    }
    
    post {
        success {
            echo "=========================================="
            echo "✅ Build ${BUILD_NUMBER} - SUCCESS"
            echo "config/config.json updated successfully"
            echo "=========================================="
        }
        failure {
            echo "=========================================="
            echo "❌ Build ${BUILD_NUMBER} - FAILED"
            echo "=========================================="
        }
    }
}
