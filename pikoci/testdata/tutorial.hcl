resource "git" "repo" {
  params {
    url = var.repo_url
    name = var.repo_name
  }
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
