resource "cron" "my_cron" {
  check_interval = "@every 10s"
  params {}
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
