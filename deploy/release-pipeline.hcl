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

resource "git" "pikoci-tag" {
  params {
    url   = "https://github.com/xescugc/pikoci"
    name  = "pikoci"
    token = var.github_token
    tag   = true
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
  get "git" "pikoci-tag" {
    trigger = true
  }
  task "docker-build-push-tag" {
    run "exec" {
      path = "/bin/sh"
      args = [
        "-ec",
        <<-EOT
        cd pikoci

        echo "${var.docker_password}" | docker login -u "${var.docker_username}" --password-stdin

        docker build -t xescugc/pikoci:$version_tag .
        docker push xescugc/pikoci:$version_tag
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

variable "github_token" {
  type = string
  secret "env" {
    key = "GITHUB_TOKEN"
  }
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
