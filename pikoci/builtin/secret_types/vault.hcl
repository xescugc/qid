secret_type "vault" {
  params = ["path"]

  get "exec" {
    path = "/bin/sh"
    args = [
      "-ec",
      <<-EOT
      vault kv get -format=json "$param_path" | jq -c '.data.data // .data'
      EOT
    ]
  }
}
