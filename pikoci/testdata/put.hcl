resource_type "git" {
  params = [
    "url",
    "name",
  ]
  check "exec" {
    path = "/bin/sh"
    args = ["-ec", "echo [{\\\"ref\\\":\\\"abc\\\"}]"]
  }
  pull "exec" {
    path = "/bin/sh"
    args = ["-ec", "echo pulling"]
  }
  push "exec" {
    path = "/bin/sh"
    args = ["-ec", "echo pushing"]
  }
}

resource "git" "repo" {
  params {
    url = "http://example.com"
    name = "repo"
  }
}

resource "cron" "timer" {
  check_interval = "@every 10s"
}

job "build-and-push" {
  get "cron" "timer" {
    trigger = true
  }
  task "build" {
    run "exec" {
      path = "echo"
      args = ["building"]
    }
  }
  put "git" "repo" {
    tag = "latest"
  }
}
