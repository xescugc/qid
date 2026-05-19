resource_type "git" {
  source = "pikoci://git"
}

resource "git" "pikoci" {
  params {
    url    = "https://github.com/xescugc/pikoci"
    name   = "pikoci"
    branch = "master"
  }
}

job "build-latest" {
  get "git" "pikoci" {
    trigger = true
  }
  task "docker-build-push-latest" {
    run "exec" {
      path = "/bin/sh"
      args = [
        "-ec",
        <<-EOT
        cd pikoci

        echo "${var.docker_password}" | docker login -u "${var.docker_username}" --password-stdin

        docker build -t xescugc/pikoci:latest .
        docker push xescugc/pikoci:latest
        EOT
      ]
    }
  }
}

job "build-release" {
  get "git" "pikoci" {
    trigger = true
    passed  = ["build-latest"]
  }
  task "docker-build-push-tag" {
    run "exec" {
      path = "/bin/sh"
      args = [
        "-ec",
        <<-EOT
        cd pikoci
        TAG=$(git describe --tags --exact-match 2>/dev/null || echo "")
        if [ -z "$TAG" ]; then
          echo "No tag found on HEAD, skipping."
          exit 0
        fi

        echo "${var.docker_password}" | docker login -u "${var.docker_username}" --password-stdin

        docker build -t xescugc/pikoci:$TAG .
        docker push xescugc/pikoci:$TAG
        EOT
      ]
    }
  }
}

secret_type "env" {
  source = "pikoci://file"
  format = "env"
  path   = "/etc/pikoci/pikoci.env"
}

variable "docker_username" {
  type = string
  secret "env" {
    key = "DOCKER_USERNAME"
  }
}

variable "docker_password" {
  type = string
  secret "env" {
    key = "DOCKER_PASSWORD"
  }
}
