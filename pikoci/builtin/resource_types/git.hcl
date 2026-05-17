resource_type "git" {
  params = [
    "url",
    "branch",
    "name",
    "token",
    "pr",
  ]
  check "exec" {
    path = "/bin/sh"
    args = [
      "-ec",
      <<-EOT
      # PR mode: check for open pull requests
      if [ "$param_pr" = "true" ]; then
        if [ -z "$param_token" ]; then
          echo "error: pr=true requires a token" >&2
          exit 1
        fi

        # GitHub PRs
        if echo "$param_url" | grep -q "github.com"; then
          REPO=$(echo "$param_url" | sed -E 's|https?://github\.com/||;s|\.git$$||')
          curl -sf -H "Authorization: token $param_token" \
            "https://api.github.com/repos/$$REPO/pulls?state=open&sort=updated&direction=desc&per_page=100" \
            | jq '[.[] | {"ref": .head.sha, "pr": (.number | tostring)}]'
          exit 0
        fi

        # GitLab MRs
        if echo "$param_url" | grep -q "gitlab.com"; then
          PROJECT=$(echo "$param_url" | sed -E 's|https?://gitlab\.com/||;s|\.git$$||' | sed 's|/|%2F|g')
          curl -sf -H "PRIVATE-TOKEN: $param_token" \
            "https://gitlab.com/api/v4/projects/$$PROJECT/merge_requests?state=opened&order_by=updated_at&sort=desc&per_page=100" \
            | jq '[.[] | {"ref": .sha, "pr": (.iid | tostring)}]'
          exit 0
        fi

        echo "error: pr=true is only supported for github.com and gitlab.com" >&2
        exit 1
      fi

      # GitHub API
      BRANCH="$${param_branch:-HEAD}"
      if [ -n "$param_token" ] && echo "$param_url" | grep -q "github.com"; then
        REPO=$(echo "$param_url" | sed -E 's|https?://github\.com/||;s|\.git$$||')
        REF=$(curl -sf -H "Authorization: token $param_token" \
          "https://api.github.com/repos/$$REPO/commits?sha=$$BRANCH&per_page=1" \
          | jq -r '.[0].sha')
        if [ -n "$$REF" ] && [ "$$REF" != "null" ]; then
          echo "[{\"ref\":\"$$REF\"}]"
          exit 0
        fi
      fi

      # GitLab API
      if [ -n "$param_token" ] && echo "$param_url" | grep -q "gitlab.com"; then
        PROJECT=$(echo "$param_url" | sed -E 's|https?://gitlab\.com/||;s|\.git$$||' | sed 's|/|%2F|g')
        REF=$(curl -sf -H "PRIVATE-TOKEN: $param_token" \
          "https://gitlab.com/api/v4/projects/$$PROJECT/repository/commits?ref_name=$$BRANCH&per_page=1" \
          | jq -r '.[0].id')
        if [ -n "$$REF" ] && [ "$$REF" != "null" ]; then
          echo "[{\"ref\":\"$$REF\"}]"
          exit 0
        fi
      fi

      # Fallback: git ls-remote
      if [ -n "$param_token" ]; then
        REF=$$(git -c credential.helper="!f() { echo password=$param_token; }; f" ls-remote "$param_url" "$$BRANCH" | awk '{print $$1}')
      else
        REF=$$(git ls-remote "$param_url" "$$BRANCH" | awk '{print $$1}')
      fi
      if [ -z "$$REF" ]; then
        REF=$$(git ls-remote "$param_url" HEAD | awk '{print $$1}')
      fi
      echo "[{\"ref\":\"$$REF\"}]"
      EOT
    ]
  }
  pull "exec" {
    path = "/bin/sh"
    args = [
      "-ec",
      <<-EOT
      # Inject token into HTTPS URL if provided
      CLONE_URL="$param_url"
      if [ -n "$param_token" ]; then
        CLONE_URL=$$(echo "$param_url" | sed -E "s|https://|https://oauth2:$param_token@|")
      fi

      if [ "$param_pr" = "true" ] && [ -n "$version_pr" ]; then
        # PR mode: fetch the PR head ref
        git clone "$$CLONE_URL" "$param_name"
        cd "$param_name"
        git fetch origin "pull/$version_pr/head:pr-$version_pr"
        git checkout "pr-$version_pr"
      elif [ -n "$param_branch" ]; then
        git clone -b "$param_branch" "$$CLONE_URL" "$param_name"
        cd "$param_name"
        git checkout "$version_ref"
      else
        git clone "$$CLONE_URL" "$param_name"
        cd "$param_name"
        git checkout "$version_ref"
      fi
      EOT
    ]
  }
  push "exec" {
    path = "/bin/sh"
    args = [
      "-ec",
      <<-EOT
      cd "$param_name"
      if [ -n "$param_token" ]; then
        REMOTE_URL=$$(echo "$param_url" | sed -E "s|https://|https://oauth2:$param_token@|")
        git remote set-url origin "$$REMOTE_URL"
      fi
      git push
      EOT
    ]
  }
}
