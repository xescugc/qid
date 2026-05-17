secret_type "file" {
  params = ["path", "format"]

  get "exec" {
    path = "/bin/sh"
    args = [
      "-ec",
      <<-EOT
      FORMAT="${param_format:-json}"
      if [ "$FORMAT" = "env" ]; then
        awk -F= 'BEGIN{printf "{"}
        /^[A-Za-z_][A-Za-z_0-9]*=/{
          key=$1; val=substr($0,index($0,"=")+1);
          gsub(/\r$/,"",val);
          gsub(/^["'"'"']|["'"'"']$/,"",val);
          gsub(/\\/,"\\\\",val); gsub(/"/,"\\\"",val);
          printf "%s\"%s\":\"%s\"",sep,key,val; sep=","
        }
        END{printf "}"}' "$param_path"
      elif [ "$FORMAT" = "json" ]; then
        cat "$param_path"
      else
        echo "ERROR: unsupported format '$FORMAT'. Use 'json' or 'env'." >&2
        exit 1
      fi
      EOT
    ]
  }
}
