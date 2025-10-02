resource_type "git" {
  inputs = [
    "url",
    "name",
  ]
  check {
    path = "/bin/sh"
    args = [
      "-ec",
      <<-EOT
        cd /
        rm -rf $NAME
        git clone --quiet $URL $NAME
        cd $NAME
        git log -5 --pretty=format:"%H"
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
        git clone --quiet $URL $NAME
        cd $NAME
        git checkout --quiet $VERSION_HASH
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
  url = "https://github.com/xescugc/qid.git"
  name = "my_repo"
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
        "/my_repo",
        "gen"
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
        "/my_repo",
        "test"
      ]
    }
  }
}
