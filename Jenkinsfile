pipeline {
    agent any

    tools { 
        go '1.18' 
    }

    stages {
        stage('Build') {
            steps {
                sh 'go get ./...'

                // Script needed to define variables
                script {
                    // Get version of program
                    VERSION = sh (
                        script: 'cat VERSION',
                        returnStdout: true
                    ).trim()

                    // Cross compile
                    sh "GOOS=linux GOARCH=amd64 go build -o ncDocConverth-${VERSION}-amd64 ./cmd/ncDocConverth"
                    sh "GOOS=linux GOARCH=arm64 go build -o ncDocConverth-${VERSION}-arm64 ./cmd/ncDocConverth"
                    sh "GOOS=windows GOARCH=arm64 go build -o ncDocConverth-${VERSION}-amd64.exe ./cmd/ncDocConverth"
                }
            }
        }

        stage('Deploy') {
            // Tags not working with gitea?
            //when {
            //    buildingTag()
            //}

            steps {
                script {
                    if (env.BRANCH_NAME == "main") {
                        sh 'sudo ./scripts/deploy.sh'
                    }
                }
            }
        }
    }

    post {
        success {
            archiveArtifacts artifacts: 'ncDocConverth-*', fingerprint: true
        }

        // Clean after build
        cleanup {
            cleanWs()
        }

        failure {
            emailext body: "${currentBuild.currentResult}: Job ${env.JOB_NAME} build ${env.BUILD_NUMBER}\n More info at: ${env.BUILD_URL}",
                recipientProviders: [[$class: 'DevelopersRecipientProvider'], [$class: 'RequesterRecipientProvider']],
                subject: "Jenkins Build ${currentBuild.currentResult}: Job ${env.JOB_NAME}"
        }
    }
}