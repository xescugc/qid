resource_type "git" {
  inputs = [
    "url",
    "name",
  ]
  check {
    path = "/bin/bash"
    args = [
      "-ec",
      <<-EOT
        git clone --quiet $URL $NAME
        cd $NAME
        if [[ -n $LAST_VERSION_HASH ]]; then
          git log $LAST_VERSION_HASH..HEAD --pretty=format:"%H"
        else
          git log -1 --pretty=format:"%H"
        fi
      EOT
    ]
  }
  pull {
    path = "/bin/sh"
    args = [
      "-ec",
      <<-EOT
        git clone $URL $NAME
        cd $NAME
        git checkout $VERSION_HASH
      EOT
    ]
  }
  push {}
}

resource "git" "repo" {
  url = var.repo_url 
  name = var.repo_name 
}

job "test" {
  get "git" "repo" {
    trigger = true
  }
  task "test" {
    run {
      path = "/bin/bash"
      args = [
        "-ec",
        <<-EOT
          cd qid_test
          go test
        EOT
      ]
    }
  }
}

job "build" {
  get "git" "repo" {
    passed = ["test"]
    trigger = true
  }
  task "build" {
    run {
      path = "/bin/bash"
      args = [
        "-ec",
        <<-EOT
          cd qid_test
          go build
        EOT
      ]
    }
  }
}

variable "repo_url" {
  type = string
}
variable "repo_name" {
  type = string
}
