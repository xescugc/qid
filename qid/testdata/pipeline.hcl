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
  push {
    path = "/bin/sh"
    args = [
      <<-EOT
        cd $NAME
        git push
      EOT
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
    on_success {
      path = "echo"
      args = [ 
        "get-s",
      ]
    }
    on_failure {
      path = "echo"
      args = [ 
        "get-f",
      ]
    }
    ensure {
      path = "echo"
      args = [ 
        "get-e",
      ]
    }
  }
  task "gen" {
    run {
      path = "make"
      args = [ 
        "-C",
        "${var.repo_name}",
        "gen"
      ]
    }
    on_success {
      path = "echo"
      args = [ 
        "task-s",
      ]
    }
    on_failure {
      path = "echo"
      args = [ 
        "task-f",
      ]
    }
    ensure {
      path = "echo"
      args = [ 
        "task-e",
      ]
    }
  }
  on_success {
    path = "echo"
    args = [ 
      "job-s",
    ]
  }
  on_failure {
    path = "echo"
    args = [ 
      "job-f",
    ]
  }
  ensure {
    path = "echo"
    args = [ 
      "job-e",
    ]
  }
}

job "notify_slack" {
  get "git" "qid" {
    trigger = true
    on_success {
      path = "echo"
      args = [ 
        "get-s",
      ]
    }
    on_failure {
      path = "echo"
      args = [ 
        "get-f",
      ]
    }
    ensure {
      path = "echo"
      args = [ 
        "get-e",
      ]
    }
  }
  task "notify" {
    run {
      path = "potato"
      args = [ 
        "slack",
      ]
    }
    on_success {
      path = "echo"
      args = [ 
        "task-s",
      ]
    }
    on_failure {
      path = "echo"
      args = [ 
        "task-f",
      ]
    }
    ensure {
      path = "echo"
      args = [ 
        "task-e",
      ]
    }
  }
  on_success {
    path = "echo"
    args = [ 
      "job-s",
    ]
  }
  on_failure {
    path = "echo"
    args = [ 
      "job-f",
    ]
  }
  ensure {
    path = "echo"
    args = [ 
      "job-e",
    ]
  }
}

job "test" {
  get "git" "qid" {
    passed  = ["gen"]
    trigger = true
    on_success {
      path = "echo"
      args = [ 
        "get-s",
      ]
    }
    on_failure {
      path = "echo"
      args = [ 
        "get-f",
      ]
    }
    ensure {
      path = "echo"
      args = [ 
        "get-e",
      ]
    }
  }
  task "test" {
    run {
      path = "make"
      args = [ 
        "-C",
        "${var.repo_name}",
        "test"
      ]
    }
    on_success {
      path = "echo"
      args = [ 
        "task-s",
      ]
    }
    on_failure {
      path = "echo"
      args = [ 
        "task-f",
      ]
    }
    ensure {
      path = "echo"
      args = [ 
        "task-e",
      ]
    }
  }
  on_success {
    path = "echo"
    args = [ 
      "job-s",
    ]
  }
  on_failure {
    path = "echo"
    args = [ 
      "job-f",
    ]
  }
  ensure {
    path = "echo"
    args = [ 
      "job-e",
    ]
  }
}

job "build" {
  get "git" "qid" {
    passed  = ["test"]
    trigger = true
    on_success {
      path = "echo"
      args = [ 
        "get-s",
      ]
    }
    on_failure {
      path = "echo"
      args = [ 
        "get-f",
      ]
    }
    ensure {
      path = "echo"
      args = [ 
        "get-e",
      ]
    }
  }
  task "build" {
    run {
      path = "make"
      args = [ 
        "-C",
        "${var.repo_name}",
        "release"
      ]
    }
    on_success {
      path = "echo"
      args = [ 
        "task-s",
      ]
    }
    on_failure {
      path = "echo"
      args = [ 
        "task-f",
      ]
    }
    ensure {
      path = "echo"
      args = [ 
        "task-e",
      ]
    }
  }
  on_success {
    path = "echo"
    args = [ 
      "job-s",
    ]
  }
  on_failure {
    path = "echo"
    args = [ 
      "job-f",
    ]
  }
  ensure {
    path = "echo"
    args = [ 
      "job-e",
    ]
  } 
}

variable "repo_url" {
  type = string
  default = "https://github.com/xescugc/qid.git"
}
variable "repo_name" {
  type = string
}
