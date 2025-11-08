resource_type "git" {
  inputs = [
    "url",
    "name",
  ]
  check "exec" {
    path = "/bin/sh"
    args = <<-EOT
        '-ec'
        'git clone --quiet $url $name
        cd $name
        if [[ -n $LAST_VERSION_HASH ]]; then
          git log $LAST_VERSION_HASH..HEAD --pretty=format:"%H"
        else
          git log -1 --pretty=format:"%H"
        fi'
      EOT
  }
  pull "exec" {
    path = "/bin/sh"
    args = <<-EOT
        '-ec'
        'git clone $url $name
        cd $name
        git checkout $VERSION_HASH'
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
  inputs {
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
