# --- Jobs ---

job "lint"{
  get "git" "pikoci_pr"{
    trigger = true
  }
  put "github-check" "ci" { status = "in_progress" }
  task "make"{
    run "docker"{
      image = "golang:1.25.1"
      cmd = "cd ${var.git_name} && make lint"
      args = [
        "-v", "pikoci-go-mod:/root/go/pkg/mod",
        "-v", "pikoci-build:/root/.cache/go-build",
      ]
    }
  }
  on_success {
    put "github-check" "ci" {
      conclusion = "success"
    }
  }

  on_failure {
    put "github-check" "ci" {
      conclusion = "failure"
    }
  }
}

job "test-mock" {
  get "git" "pikoci_pr"{
    trigger = true
  }
  put "github-check" "ci" { status = "in_progress" }
  task "make"{
    run "docker"{
      image = "golang:1.25.1"
      cmd = "cd ${var.git_name} && make test-mock"
    }
  }
  on_success {
    put "github-check" "ci" {
      conclusion = "success"
    }
  }

  on_failure {
    put "github-check" "ci" {
      conclusion = "failure"
    }
  }
}

job "test-integration" {
  get "git" "pikoci_pr"{
    trigger = true
  }
  put "github-check" "ci" { status = "in_progress" }
  task "make"{
    run "docker"{
      image = "golang:1.25.1"
      cmd = "cd ${var.git_name} && make test-integration"
      args = [
        "-v", "pikoci-go-mod:/root/go/pkg/mod",
        "-v", "pikoci-build:/root/.cache/go-build",
      ]
    }
  }
  on_success {
    put "github-check" "ci" {
      conclusion = "success"
    }
  }

  on_failure {
    put "github-check" "ci" {
      conclusion = "failure"
    }
  }
}

job "test-backends" {
  get "git" "pikoci_pr"{
    trigger = true
    passed = ["lint", "test-mock", "test-integration"]
  }

  put "github-check" "ci" {
    status = "in_progress"
  }

  service "mariadb" {}
  service "postgresql" {}
  service "nats" {}
  service "rabbitmq" {}
  service "kafka" {}
  service "vault" {}

  task "make"{
    run "docker"{
      image = "golang:1.25.1"
      cmd = "cd ${var.git_name} && make test-backends"
      args = [
        "--network=host",
        "-v", "pikoci-go-mod:/root/go/pkg/mod",
        "-v", "pikoci-build:/root/.cache/go-build",
      ]
    }
  }

  on_success {
    put "github-check" "ci" {
      conclusion = "success"
    }
  }

  on_failure {
    put "github-check" "ci" {
      conclusion = "failure"
    }
  }
}

# --- Resources, Secrets and Variables ---
variable "git_url"{
  type = string
  default = "https://github.com/xescugc/pikoci"
}

variable "git_name"{
  type = string
  default = "pikoci"
}

variable "timeout" {
  type = string
  default = "5m"
}

secret_type "env" {
  source = "pikoci://file"
  format = "env"
  path = "/etc/pikoci/pikoci.env"
}

variable "github_token" {
  type = string
  secret "env"{
    key = "GITHUB_TOKEN"
  }
}

variable "github_app_id" {
  type = string
  secret "env"{
    key = "GITHUB_APP_ID"
  }
}

variable "github_app_installation_id" {
  type = string
  secret "env"{
    key = "GITHUB_APP_INSTALLATION_ID"
  }
}


secret_type "pickoci_github_pem" {
  source = "pikoci://file"
  format = "raw"
  path = "/etc/pikoci/pikoci_github_app.pem"
}

variable "pikoci_github_app_pem" {
  type = string
  secret "pickoci_github_pem" {
    key = "content"
  }
}

resource_type "git"{
  source = "pikoci://git"
}

resource "git" "pikoci_pr"{
  params {
    url = var.git_url
    name= var.git_name
    pr = true
    token = var.github_token
  }
}

resource_type "github-check" {
  source = "https://github-check"
}

resource "github-check" "ci" {
  params {
    app_id          = var.github_app_id
    installation_id = var.github_app_installation_id
    private_key     = var.pikoci_github_app_pem
    repository      = "xescugc/pikoci"
  }
}


# --- Services ---

service_type "mariadb" {
  start "exec" {
    path = "/bin/sh"
    args = ["-ec", "docker run -d --name mariadb-$BUILD_ID --network=host -e MYSQL_ROOT_PASSWORD=root123 mariadb:11.4.2"]
  }
  ready_check "exec" {
    path     = "/bin/sh"
    args     = ["-ec", "docker exec mariadb-$BUILD_ID mariadb -uroot -proot123 -e 'SELECT 1'"]
    interval = "2s"
    timeout  = var.timeout
  }
  stop "exec" {
    path = "/bin/sh"
    args = ["-ec", "docker rm -f mariadb-$BUILD_ID"]
  }
}

service_type "postgresql" {
  start "exec" {
    path = "/bin/sh"
    args = ["-ec", "docker run -d --name pg-$BUILD_ID --network=host -e POSTGRES_PASSWORD=postgres123 postgres:17"]
  }
  ready_check "exec" {
    path     = "/bin/sh"
    args     = ["-ec", "docker exec pg-$BUILD_ID pg_isready"]
    interval = "2s"
    timeout  = var.timeout
  }
  stop "exec" {
    path = "/bin/sh"
    args = ["-ec", "docker rm -f pg-$BUILD_ID"]
  }
}

service_type "nats" {
  start "exec" {
    path = "/bin/sh"
    args = ["-ec", "docker run -d --name nats-$BUILD_ID --network=host nats:2.12.0"]
  }
  ready_check "exec" {
    path     = "/bin/sh"
    args     = ["-ec", "nc -z localhost 4222"]
    interval = "1s"
    timeout  = var.timeout
  }
  stop "exec" {
    path = "/bin/sh"
    args = ["-ec", "docker rm -f nats-$BUILD_ID"]
  }
}

service_type "rabbitmq" {
  start "exec" {
    path = "/bin/sh"
    args = ["-ec", "docker run -d --name rabbit-$BUILD_ID --network=host -e RABBITMQ_NODENAME=rabbit rabbitmq:3"]
  }
  ready_check "exec" {
    path     = "/bin/sh"
    args     = ["-ec", "nc -z localhost 5672"]
    interval = "3s"
    timeout  = var.timeout
  }
  stop "exec" {
    path = "/bin/sh"
    args = ["-ec", "docker rm -f rabbit-$BUILD_ID"]
  }
}

service_type "kafka" {
  start "exec" {
    path = "/bin/sh"
    args = ["-ec", "docker run -d --name kafka-$BUILD_ID --network=host -e KAFKA_NODE_ID=0 -e KAFKA_PROCESS_ROLES=controller,broker -e KAFKA_CONTROLLER_QUORUM_VOTERS=0@localhost:9093 -e KAFKA_LISTENERS=PLAINTEXT://:9092,CONTROLLER://:9093 -e KAFKA_ADVERTISED_LISTENERS=PLAINTEXT://127.0.0.1:9092 -e KAFKA_CONTROLLER_LISTENER_NAMES=CONTROLLER -e KAFKA_LISTENER_SECURITY_PROTOCOL_MAP=CONTROLLER:PLAINTEXT,PLAINTEXT:PLAINTEXT -e CLUSTER_ID=MkU3OEVBNTcwNTJENDM2Qk apache/kafka:latest"]
  }
  ready_check "exec" {
    path     = "/bin/sh"
    args     = ["-ec", "nc -z localhost 9092"]
    interval = "3s"
    timeout  = var.timeout
  }
  stop "exec" {
    path = "/bin/sh"
    args = ["-ec", "docker rm -f kafka-$BUILD_ID"]
  }
}

service_type "vault" {
  start "exec" {
    path = "/bin/sh"
    args = ["-ec", "docker run -d --name vault-$BUILD_ID --network=host -e VAULT_DEV_ROOT_TOKEN_ID=test-root-token -e VAULT_DEV_LISTEN_ADDRESS=0.0.0.0:8200 -e SKIP_SETCAP=1 --cap-add IPC_LOCK hashicorp/vault:latest"]
  }
  ready_check "exec" {
    path     = "/bin/sh"
    args     = ["-ec", "curl -sf http://127.0.0.1:8200/v1/sys/health"]
    interval = "2s"
    timeout  = var.timeout
  }
  stop "exec" {
    path = "/bin/sh"
    args = ["-ec", "docker rm -f vault-$BUILD_ID"]
  }
}
