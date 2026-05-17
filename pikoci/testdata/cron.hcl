variable "secrets_file" {
  type    = string
  default = "pikoci/testdata/secrets.json"
}

secret_type "my-file" {
  source = "pikoci://file"
}

variable "greeting" {
  type = string
  secret "my-file" {
    path = var.secrets_file
    key  = "greeting"
  }
}

variable "env" {
  type = string
  secret "my-file" {
    path = var.secrets_file
    key  = "env"
  }
}

resource "cron" "my_cron" {
  check_interval = "@every 10s"
}

job "gen" {
  get "cron" "my_cron" {
    trigger = true
  }
  task "echo" {
    run "exec" {
      path = "echo"
      args = ["greeting=${var.greeting} env=${var.env}"]
    }
  }
}
