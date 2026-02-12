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

        stage('Update Build Number') {
            steps {
                script {
                    echo "Updating build number for provision_id: ${PROVISION_ID}"
                    echo "Current BUILD_NUMBER: ${BUILD_NUMBER}"

                    // Read config.json
                    def configFile = readFile file: "${CONFIG_FILE}"
                    def config = new groovy.json.JsonSlurper().parseText(configFile)

                    // Find the target provision entry
                    def targetEntry = config.config_specs.find { it.provision_id == "${PROVISION_ID}" }

                    if (targetEntry) {
                        def currentNumber = targetEntry.build_number.number
                        echo "Current minor version in config: ${currentNumber}"

                        // Update minor version to BUILD_NUMBER
                        targetEntry.build_number.number = "${BUILD_NUMBER}"
                        echo "Updated minor version: ${currentNumber} -> ${BUILD_NUMBER}"

                        // Write back to config.json
                        writeFile file: "${CONFIG_FILE}", text: groovy.json.JsonOutput.toJson(config)
                        // Pretty print JSON
                        sh "jq '.' ${CONFIG_FILE} > ${CONFIG_FILE}.tmp && mv ${CONFIG_FILE}.tmp ${CONFIG_FILE}"

                        echo "Build number updated successfully to: ${BUILD_NUMBER}"
                    } else {
                        error("Provision ID ${PROVISION_ID} not found in config!")
                    }
                }
            }
        }

        stage('Commit Changes') {
            steps {
                script {
                    sh """
                        git config user.email "jenkins@hynix.local"
                        git config user.name "Jenkins Agent"
                        git add ${CONFIG_FILE}
                        git commit -m "Update build_number to \${BUILD_NUMBER}"
                    """
                }
            }
        }

        stage('Push Changes') {
            steps {
                script {
                    withCredentials([usernamePassword(
                        credentialsId: "${GIT_CREDENTIALS_ID}",
                        usernameVariable: 'GIT_USERNAME',
                        passwordVariable: 'GIT_PASSWORD'
                    )]) {
                        sh """
                            git push https://\${GIT_USERNAME}:\${GIT_PASSWORD}@github.com/philllipjung/hynix.git master
                        """
                    }
                }
            }
        }
    }

    post {
        success {
            echo "Pipeline completed successfully - minor version updated to ${BUILD_NUMBER}"
        }
        failure {
            echo "Pipeline failed!"
        }
    }
}
