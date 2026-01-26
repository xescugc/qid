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

runner "docker" {
  run {
    path = "docker"
    args = [
      "run", "--rm",
      "-v","$WORKDIR:/workdir",
      "-w","/workdir",
      "$image",
      "$cmd"
    ]
  }
}

resource "git" "qid" {
  params {
    url = var.repo_url 
    name = "${var.repo_name}"
  }
}

job "gen" {
  get "git" "qid" {
    trigger = true
  }
  task "gen" {
    run "docker" {
      image = "golang:1.25.1"
      cmd = "make -C ${var.repo_name} gen"
    }
  }
}

job "test" {
  get "git" "qid" {
    passed  = ["gen"]
    trigger = true
  }
  task "test" {
    run "docker" {
      image = "golang:1.25.1"
      cmd = "make -C ${var.repo_name} test"
    }
  }
}

job "build" {
  get "git" "qid" {
    passed  = ["test"]
    trigger = true
  }
  task "build" {
    run "exec" {
      path = "make"
      args = "-C ${var.repo_name} release"
    }
  }
}

variable "repo_url" {
  type = string
  default = "https://github.com/xescugc/qid.git"
}

variable "repo_name" {
  type = string
}
