runner "docker" {
  run {
    path = "docker"
    args = [
      "run", "--rm",
      "-v", "$WORKDIR:/workdir",
      "-w", "/workdir",
      "$args",
      "$image",
      "/bin/sh", "-ec", "$cmd",
    ]
  }
}
