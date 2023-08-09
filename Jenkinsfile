// Base class containing the git configuration
class Configuration {

	// (Long) Hash value of the current commit
	String commitHash;
	// Name of the current Branch
	String branch;

	// The last available tag in the git commit history
	String lastTag;
	// Tag value of the current commit
	String[] currentTags;

	// Tag used to update the helm values
	String updateTag = ""
    String updateFile = ""
    String updateFile2 = ""

	// Tags returns the tags to apply for the build container image.
	def Tags() {
		ArrayList rtc = []

		// Building on master branch
		if (branch == "master" || branch == "main") {
			// When building on the master branch always use the provided tags for the current commit
			if (currentTags != null) rtc.addAll(currentTags)
			
			// The master branch is used for "release candidate" and "production build"
			currentTags.each {
				if (it.contains("-rc.")) {
					// Update tag "latest-rc" because a tag like "v1.10.0-rc.1" was provided 
					rtc << "rc-latest"
					updateTag = it
                    updateFile = "rc"
				} else {
					// The tag is not a rc -> new "production" release
					rtc << "latest"
					updateTag = it
                    updateFile = "main"
                    // Also update the rc when "main" was updated
                    updateFile2 = "rc"
				}
			}

			// Also push a tag with the current commit hash
			rtc << "main-" + commitHash
		} else if (branch == "snapshot") {
			// For security reasons the tags on the snapshot branch are not used for tagging
			rtc << "snapshot-latest"

			// Otherwise only push a tag with the current commit hash
			rtc << "snapshot-" + commitHash

			updateTag =  "snapshot-" + commitHash
            updateFile = "snapshot"
		} else {
            currentBuild.result = 'ABORTED'
            error("Received not supported branch for building container image: '" + branch + "'")
		}

		return rtc
	}

	// Returns the current version of the program that should be used during building
	String Version() {
		if (branch == "master" || branch == "main") {
			return lastTag
		} else {
			// Otherwiese add a "-snapshot" to the last tagged version
			return "" + lastTag.replace("(?<=v\\d{1,3}\\.\\d{1,3}\\.\\d{1,3}).*", "") + "-dev"
		}
	}

}

// Global variables
def Configuration gitConfig = new Configuration()
def String VERSION

// App used for helm identification and commit message
def String APP_NAME = "ncDocConverter"

pipeline {

    agent {
        // Use the kubernetes agent
        kubernetes { 
            label 'podman-low'
        }
    }

    stages {

        stage('Initializing variables') {
            steps {
                script {
					gitConfig.commitHash = "${env.GIT_COMMIT}"
					gitConfig.branch = "${env.GIT_BRANCH}"

					currentTags = sh (
						script: 'git tag --points-at HEAD',
						returnStdout: true
					)
					if (currentTags != "") {
						gitConfig.currentTags = currentTags.split("\n")
					}

					gitConfig.lastTag = sh (
						script: 'git describe --tags --abbrev=0',
						returnStdout: true
					).replace("\n", "")

					// Apply the current version code
					VERSION = gitConfig.Version()
                }
            }
        }

        stage('Build') {
            steps {
                echo "Building Version '${VERSION}' and tagging it with '${gitConfig.Tags()}'"
                container('podman-low') {
                    script {

                        withEnv([ "version=${VERSION}", "commit=${gitConfig.commitHash}" ]) {
                            sh 'buildah bud --layers --build-arg VERSION="${version}" --tag=rpjosh.de/jenkins-ncdocconverter:${commit} \
                                    --cache-to=git.rpjosh.de/build-cache/ncdocconverter --cache-from=git.rpjosh.de/build-cache/ncdocconverter \
                                    -f Dockerfile .'
                        }
                    }
                }

            }
        }

        stage('Publish') {
            steps {
                echo "Publish to docker repository (git.rpjosh.de/rpdb)"
                container('podman-low') {
                    script {
                        gitConfig.Tags().each {
                            sh "buildah push rpjosh.de/jenkins-ncdocconverter:${gitConfig.commitHash} docker://git.rpjosh.de/rpjosh-container/ncdocconverter:${it}"
                        }
                    }
                }
            }
        }

        stage('Deploy') {
            steps {
              
                container('podman-low') {
                    script {

                        configFileProvider([configFile(fileId: 'deployConfig', variable: 'confFile')]) {
                            script {
                                // Read the kubernets deploy configuration from the file
                                def config = readJSON file:"$confFile"
                                def String url = config.kubernetes.gitHelmValues + APP_NAME + "/" + gitConfig.updateFile + ".yaml"
                                def String url2 = config.kubernetes.gitHelmValues + APP_NAME + "/" + gitConfig.updateFile2 + ".yaml"
                                
                                sh "echo Using git URL '${url}'"
                                // Get the current file content, replace the tag, and push the mmodified tag again
                                withCredentials([ string(credentialsId: 'GIT_API_KEY', variable: "gitApiKey") ]) {
                                    withEnv([ "url=${url}", "url2=${url2}", "tag=${gitConfig.updateTag}", "file=${gitConfig.updateFile}", "app=${APP_NAME}" ]) {
                                        // Exit when one command does fail in pipe
                                        sh "set -e && set -o pipefail"

                                        if (gitConfig.updateFile == "main") {
                                            echo "Updating git configuration dirctory (${APP_NAME}/${gitConfig.updateFile}.yaml) with tag [${gitConfig.updateTag}]"

                                            sh 'curl -s "https://notNeeded:${gitApiKey}@${url}" | jq -r .content | base64 --decode > tmp_values.yaml'
                                            sh 'curl -s --fail-with-body "https://notNeeded:${gitApiKey}@${url}" -X PUT -H "Content-Type: application/json" -d \
                                                \'{ "content": "\'"$(cat tmp_values.yaml | sed -e \'s/tag: ".*"/tag: "\'$tag\'"/g\' | base64 -w 0)"\'", "message": "[CI] Update image for \'"$app-$file"\'", \
                                                    "sha": "\'$(git hash-object tmp_values.yaml | tr -d "\\n")\'" }\' '
                                        }
                                    }
                                }
                            }
                        }
                    }
                }
            }
        }
    }

    post {
        success {
            sh 'echo Build finished'
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