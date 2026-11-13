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
