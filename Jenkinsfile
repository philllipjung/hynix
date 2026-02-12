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
                        echo "Current build_number in config: ${currentNumber}"

                        // Check if current format is decimal (e.g., "4.0.1")
                        def newNumber
                        if (currentNumber =~ /^\d+\.\d+\.\d+$/) {
                            // Parse version parts: major.minor.patch
                            def parts = currentNumber.split('\\.')
                            def major = parts[0]
                            def patch = parts[2]
                            // Replace middle part with BUILD_NUMBER
                            newNumber = "${major}.${BUILD_NUMBER}.${patch}"
                            echo "Decimal format detected. Updating: ${currentNumber} -> ${newNumber}"
                        } else {
                            // Non-decimal format, just use BUILD_NUMBER as-is
                            newNumber = "${BUILD_NUMBER}"
                            echo "Non-decimal format. Setting to: ${newNumber}"
                        }

                        // Update the config
                        targetEntry.build_number.number = newNumber

                        // Write back to config.json
                        writeFile file: "${CONFIG_FILE}", text: groovy.json.JsonOutput.toJson(config)
                        // Pretty print the JSON
                        sh "jq '.' ${CONFIG_FILE} > ${CONFIG_FILE}.tmp && mv ${CONFIG_FILE}.tmp ${CONFIG_FILE}"

                        echo "Updated build_number to: ${newNumber}"
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
            echo "Build number updated successfully to ${BUILD_NUMBER}"
        }
        failure {
            echo "Pipeline failed!"
        }
    }
}
