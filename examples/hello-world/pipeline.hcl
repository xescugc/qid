resource "cron" "tick" {
  check_interval = "@every 10s"
}

job "hello" {
  get "cron" "tick" {
    trigger = true
  }
  task "echo" {
    run "exec" {
      path = "echo"
      args = ["Hello from PikoCI!"]
    }
  }
}
