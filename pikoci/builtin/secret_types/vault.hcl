secret_type "vault" {
  params = ["path", "address", "token"]

  get "exec" {
    path = "/bin/sh"
    args = [
      "-ec",
      <<-EOT
      VAULT_ADDR=$param_address VAULT_TOKEN=$param_token vault kv get -format=json "$param_path" | jq -c '.data.data // .data'
      EOT
    ]
  }
}
