pipeline {
    agent {
        docker {
            image 'node:9.4'
            args '-p 3000:3000'
        }
    }
    environment {
        CI = 'true'
    }
    stages {
        stage('Build') {
            steps {
                sh 'npm install'
            }
        }
        stage('Staging') {
            steps {
                sh './jenkins/scripts/testing.sh'
            }
        }
        stage('Production') {
            steps {
                sh './jenkins/scripts/production.sh'
                input message: 'Stop the production server'
                sh './jenkins/scripts/stop.sh'
            }
        }
    }
}
