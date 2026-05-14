resource_type "git" {
  params = [
    "url",
    "name",
  ]
  check "exec" {
    path = "/bin/sh"
    args = <<-EOT
        '-ec'
        'git clone --quiet $param_url $param_name
        cd $param_name
        if [[ -n $version_ref ]]; then
          git log $version_ref..HEAD --pretty=format:"%H" | jq -Rsc "(. / \"\n\" | map(select(length>0) | { "ref": . }))"
        else
          git log -1 --pretty=format:"%H" | jq -Rsc "(. / \"\n\" | map(select(length>0) | { "ref": . }))"
        fi'
      EOT
  }
  pull "exec" {
    path = "/bin/sh"
    args = <<-EOT
        '-ec'
        'git clone $param_url $param_name
        cd $param_name
        git checkout $version_ref'
      EOT
  }
  push "exec" { }
}

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
