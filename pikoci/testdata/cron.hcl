variable "secrets_file" {
  type    = string
  default = "pikoci/testdata/secrets.json"
}

secret_type "my-file" {
  source = "pikoci://file"
}

secret "my-file" "app-secrets" {
  path = var.secrets_file
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
    secrets = ["my-file.app-secrets"]
    run "exec" {
      path = "echo"
      args = ["greeting=$secret_greeting env=$secret_env"]
    }
  }
}
