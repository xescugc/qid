secret_type "file" {
  params = ["path"]

  get "exec" {
    path = "/bin/sh"
    args = [
      "-ec",
      "cat \"$param_path\""
    ]
  }
}
