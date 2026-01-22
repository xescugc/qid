resource "cron" "my_cron" {
  check_interval = "@every 5s"
  inputs {}
}

job "gen" {
  get "cron" "my_cron" {
    trigger = true
  }
  task "echo" {
    run "exec" {
      path = "echo"
      args = "'IN'"
    }
  }
}
