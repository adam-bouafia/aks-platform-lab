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

## 2026-11-13 CI failed: trivy-action's own pinned version was broken upstream

Symptom:     After fixing the setup-tflint SHA, CI failed again in a different step:
             Error: Unable to resolve action `aquasecurity/setup-trivy@v0.2.1`, unable
             to find version `v0.2.1`. That action is not referenced anywhere in
             ci.yml directly.
Looked at:   Fetched aquasecurity/trivy-action's own action.yaml at the pinned commit
             (v0.28.0) and found it internally calls `uses: aquasecurity/setup-trivy@v0.2.1`
             as a nested step. `git ls-remote --tags` on aquasecurity/setup-trivy showed
             the oldest tag still present is v0.2.6 - v0.2.1 had been deleted/retagged
             upstream. Checked the latest trivy-action release (v0.36.0) and confirmed
             its action.yaml now pins setup-trivy to a commit SHA for v0.2.6, which
             does exist.
Root cause:  Our SHA pin for trivy-action@0.28.0 was valid, but that specific old
             release of trivy-action itself has a broken internal dependency (an
             upstream tag it references no longer exists). Pinning by SHA protects
             against the tag moving, but not against the pinned release depending on
             something that later disappears.
Fix:         Upgraded the trivy-action pin to v0.36.0 (SHA a9c7b0f06e461e9d4b4d1711f154ee024b8d7ab8),
             whose own internal setup-trivy reference is healthy.
Prevent:     When a pinned action fails to resolve a *different* action than the one
             named in the failing step, suspect a nested/composite action dependency.
             Check the failing action's own action.yaml/action.yml at the pinned SHA.
             Prefer reasonably recent releases of third-party actions, since very old
             pinned versions can depend on upstream refs that get cleaned up over time.

## 2026-11-13 CI failed: sbom-action's "path" input double-prefixed the scan target

Symptom:     Generate CycloneDX SBOM job failed with "Error: The process
             '/opt/hostedtoolcache/syft/1.19.0/x64/syft' failed with exit code 1".
             The printed command showed the real problem:
             `syft scan dir:docker-archive:/tmp/image.tar -o cyclonedx-json`
             - two scheme prefixes glued together, which Syft can't parse.
Looked at:   The workflow passed `path: docker-archive:/tmp/image.tar` to
             anchore/sbom-action. Checked the action's README: `path` is documented
             as "a path on the filesystem to scan", separate from `image` ("a
             container image to scan"). Downloaded the action's compiled
             dist/runSyftAction/index.js at the pinned SHA and found the exact
             branching: the `image` input is passed to syft unprefixed, `file` gets
             `file:` prepended, and `path` always gets `dir:` prepended - regardless
             of what string you put in it.
Root cause:  Used the wrong input. `path` is meant for a plain directory and always
             gets `dir:` glued on the front, so giving it an already-complete
             `docker-archive:...` scheme produced an invalid double-prefixed target.
Fix:         Switched the step to use `image: docker-archive:/tmp/image.tar` instead
             of `path:` - that input is forwarded to syft as-is, which is what a
             scheme-prefixed scan target needs.
Prevent:     When a wrapper action's documented inputs don't make the exact command
             it will run obvious, read its actual source (the compiled dist/*.js for
             a Node action) rather than guessing from the README's short description
             alone - the parameter names can look interchangeable but build very
             different commands underneath.
