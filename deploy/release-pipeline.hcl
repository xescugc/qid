resource_type "git" {
  source = "pikoci://git"
}

resource_type "git-tag" {
  params = [
    "url",
    "name",
    "token",
  ]
  check "exec" {
    path = "/bin/sh"
    args = [
      "-ec",
      <<-EOT
      URL="$param_url"
      TOKEN="$param_token"

      REPO=$(echo "$URL" | sed -E 's|https?://github\.com/||;s|\.git$||')
      TAG=$(curl -sf -H "Authorization: token $TOKEN" \
        "https://api.github.com/repos/$REPO/tags?per_page=1" \
        | jq -r '.[0].name // empty')

      if [ -z "$TAG" ]; then
        echo "[]"
        exit 0
      fi

      SHA=$(curl -sf -H "Authorization: token $TOKEN" \
        "https://api.github.com/repos/$REPO/git/ref/tags/$TAG" \
        | jq -r '.object.sha')

      echo "[{\"ref\":\"$SHA\",\"tag\":\"$TAG\"}]"
      EOT
    ]
  }
  pull "exec" {
    path = "/bin/sh"
    args = [
      "-ec",
      <<-EOT
      URL="$param_url"
      TOKEN="$param_token"
      TAG="$version_tag"

      if [ -n "$TOKEN" ]; then
        URL=$(echo "$URL" | sed -E "s|https://|https://oauth2:$${TOKEN}@|")
      fi

      git clone -b "$TAG" --depth 1 "$URL" "$param_name"
      EOT
    ]
  }
  push "exec" { }
}

resource "git" "pikoci" {
  params {
    url    = "https://github.com/xescugc/pikoci"
    name   = "pikoci"
    branch = "master"
  }
}

resource "git-tag" "pikoci-tag" {
  params {
    url   = "https://github.com/xescugc/pikoci"
    name  = "pikoci"
    token = var.github_token
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
  get "git-tag" "pikoci-tag" {
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
