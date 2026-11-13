# Failures log

Format: Symptom / Looked at / Root cause / Fix / Prevent

## 2026-11-13 Docker daemon not running

Symptom:     `docker build`, `docker images`, `docker run` and `trivy image` all failed
             with "Cannot connect to the Docker daemon at unix:///var/run/docker.sock.
             Is the docker daemon running?" (plus noisy fallback errors from trivy
             trying containerd, podman and Docker Hub).
Looked at:   `systemctl is-active docker` -> inactive; `systemctl is-enabled docker` ->
             disabled; `groups $USER` -> already includes `docker`, so permissions were
             not the problem.
Root cause:  The docker service was installed but never enabled or started on this
             machine.
Fix:         `sudo systemctl enable --now docker`
Prevent:     When any docker/trivy command fails with a socket-connection error, check
             `systemctl is-active docker` first, before assuming it's a build or image
             problem.

## 2026-11-13 hadolint "file does not exist"

Symptom:     `hadolint app/Dockerfile.naive` and `hadolint app/Dockerfile` both failed
             with `withBinaryFile: does not exist (No such file or directory)`.
Looked at:   `pwd` showed the shell was already inside `app/`, and `ls Dockerfile*`
             confirmed the files are at `app/Dockerfile` and `app/Dockerfile.naive`,
             not `app/app/Dockerfile*`.
Root cause:  Path doubled up: ran the command with an `app/` prefix while already
             inside the `app/` directory, so hadolint looked for a path that does
             not exist.
Fix:         Run hadolint with the path relative to the current directory
             (`hadolint Dockerfile.naive`, `hadolint Dockerfile`) instead of
             repeating the `app/` prefix.
Prevent:     Check `pwd` before pasting a path from higher up the repo tree into a
             command run from inside a subdirectory.

## 2026-11-13 CI failed: unable to resolve pinned action SHA

Symptom:     GitHub Actions run failed immediately in "Prepare all required actions"
             with: Error: Unable to resolve action
             `terraform-linters/setup-tflint@99297851292694f0418c5f08843f29a853c529a1`,
             unable to find version `99297851292694f0418c5f08843f29a853c529a1`.
Looked at:   `git ls-remote --tags https://github.com/terraform-linters/setup-tflint.git`
             to list the real commit SHA behind the v4.1.1 tag; it did not match what
             was in ci.yml. Checked all 12 pinned actions in the file the same way
             (for annotated tags, the real commit is the `^{}` peeled ref, not the
             tag object itself) and found 6 wrong: setup-tflint, trivy-action,
             codeql-action/upload-sarif, download-artifact, sbom-action, login-action.
Root cause:  Several commit SHAs pinned in the workflow did not correspond to the
             release tag named in their comment - likely fabricated/guessed rather
             than copied from the real repo (one, docker/login-action, was off by
             only a few characters from the real SHA, easy to miss on a visual read).
Fix:         Looked up the real commit for each tag with
             `git ls-remote --tags https://github.com/<owner>/<repo>.git` and replaced
             every wrong SHA with the verified one.
Prevent:     Never trust a generated `@<sha> # <tag>` pin at face value. Verify every
             pin with `git ls-remote --tags` (using the `^{}` peeled SHA for annotated
             tags) before committing a workflow file, especially for less common
             actions.
