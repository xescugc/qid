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
        cd /
        rm -rf $NAME
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
        cd /
        rm -rf $NAME
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

resource "git" "my_repo" {
  url = var.repo_url 
  name = "my_${var.repo_name}"
}

job "gen" {
  get "my_repo" {
    trigger = true
  }
  task "gen" {
    run {
      path = "make"
      args = [ 
        "-C",
        "/my_potato",
        "gen"
      ]
    }
  }
}

job "notify_slack" {
  get "my_repo" {
    trigger = true
  }
  task "notify" {
    run {
      path = "potato"
      args = [ 
        "slack",
      ]
    }
  }
}

job "test" {
  get "my_repo" {
    passed  = ["gen"]
    trigger = true
  }
  task "test" {
    run {
      path = "make"
      args = [ 
        "-C",
        "/my_potato",
        "test"
      ]
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
