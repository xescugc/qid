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
      args = ["-C", "${var.repo_name}", "release"]
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
