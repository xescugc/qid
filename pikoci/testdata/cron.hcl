variable "secrets_file" {
  type    = string
  default = "pikoci/testdata/secrets.json"
}

secret_type "my-file" {
  source = "pikoci://file"
}

resource "cron" "my_cron" {
  check_interval = "@every 10s"
  params {}
}

job "gen" {
  get "cron" "my_cron" {
    trigger = true
  }
  task "echo" {
    secrets = {
      "my-file" = var.secrets_file
    }
    run "exec" {
      path = "echo"
      args = ["greeting=$secret_greeting env=$secret_env"]
    }
  }
}
