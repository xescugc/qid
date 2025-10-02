job "echo1" {
  task "echo" {
    run {
      path = "echo"
      args = [ "potato 1" ]
    }
  }
}

job "echo2" {
  get "echo1" {
    passed  = ["job.echo1"]
    trigger = true
  }
  task "ehco" {
    run {
      path = "echo"
      args = [ "potato 2 " ]
    }
  }
}
