resource_type "cron" {
  check "exec" {
    path = "/bin/sh"
    args = [
      "-ec",
      <<-EOT
      echo "[{\"date\":\"$(date)\"}]"
      EOT
    ]
  }
  pull "exec" { }
  push "exec" { }
}
