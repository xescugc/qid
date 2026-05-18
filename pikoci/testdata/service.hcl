service_type "test-db" {
  params = ["version"]

  start "exec" {
    path = "/bin/sh"
    args = ["-ec", "echo starting db version=$param_version"]
  }

  ready_check "exec" {
    path  = "/bin/sh"
    args  = ["-ec", "echo ready"]
    interval = "1s"
    timeout  = "10s"
  }

  stop "exec" {
    path = "/bin/sh"
    args = ["-ec", "echo stopping db"]
  }
}

resource "cron" "timer" {
  check_interval = "@every 1h"
}

job "with-service-ref" {
  service "test-db" {}

  get "cron" "timer" {
    trigger = true
  }
  task "run-tests" {
    run "exec" {
      path = "echo"
      args = ["testing"]
    }
  }
}

job "with-service-params" {
  service "test-db" {
    version = "15"
  }

  get "cron" "timer" {
    trigger = true
  }
  task "run-tests" {
    run "exec" {
      path = "echo"
      args = ["testing with params"]
    }
  }
}


