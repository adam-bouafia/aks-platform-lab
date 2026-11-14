# End-of-phase Q&A checkpoints

Format: 3 questions per phase, Adam's answer, then the corrected/complete answer.

## Phase 1 — hardened Go app + Dockerfile

Q1: What is `/healthz` for, and who calls it?
A (Adam): checks if the Kubernetes process is alive or not.
Corrected: it's the liveness probe target — kubelet calls it to decide whether to
restart the container. It should only fail if the process itself is unrecoverable
(deadlocked, corrupted state), never because a dependency is slow.

Q2: What is `/readyz` for, and how does this app implement it?
A (Adam): checks the secret file.
Corrected: readiness probe — kubelet/Service use it to decide whether to route
traffic to this pod. In `ledger-api` it checks that the secret file at `secretPath`
exists and is readable, since the app can't serve `/api/v1/transactions` without it.
Liveness answers "is the process alive", readiness answers "can it serve traffic
right now" — different questions, different consequences (restart vs. remove from
Service endpoints).

Q3: Why does the final image run as `USER 65532:65532` even though it's based on
`distroless`?
A (Adam): distroless has root 0, so we don't need nonroot since distroless doesn't
need root permission.
Corrected: this was wrong and got corrected in session — distroless images default
to **root (UID 0)** unless you explicitly use the `:nonroot` tag or set `USER`.
"Distroless" only means no shell/package manager/OS userland, not that it's
automatically unprivileged. `gcr.io/distroless/static-debian12:nonroot` already
defaults to UID 65532, but the Dockerfile sets `USER 65532:65532` explicitly so
it's documented and enforced regardless of base image changes. Running as non-root
limits blast radius if the app is ever compromised — no write access to most of the
filesystem, can't bind privileged ports, can't install anything (no shell anyway).

(Third Phase 1 question's exact original wording and Adam's answer about an "attack
name" he didn't recall — reconstructed from session summary, not verbatim. Topic was
`subtle.ConstantTimeCompare` in the API key check: it prevents a **timing attack**,
where an attacker measures response-time differences in a naive `==` string compare
to guess the secret byte-by-byte. `ConstantTimeCompare` always takes the same time
regardless of where the mismatch occurs.)

## Phase 2 — CI/CD pipeline

Q1: Your workflow has `needs: [lint-test, iac-scan]` gating `build-scan`, and pins
every action by full commit SHA. If someone opens a PR from a fork, what permission
does `secrets.GITHUB_TOKEN` have in that run by default, and why does that matter
for the `push` job's `if` condition (`github.ref == 'refs/heads/main' &&
github.event_name == 'push'`)?
A (Adam): GITHUB_TOKEN is like the key access for CI to run the tests; if someone
besides admin opens a PR it obviously won't get pushed to main.
Corrected: on a fork PR, GitHub **forces** `GITHUB_TOKEN` to read-only, no matter
what the workflow's `permissions:` block requests — a hard platform rule, not a
config choice. But the real reason `push` can't run on any PR (fork or not) is the
`if` condition itself: a PR is event `pull_request`, never `push`, so `push` is
structurally excluded regardless of who opened it or what token scope they'd have.
The fork token restriction is a second, independent safety net in case that
condition were ever written wrong.

Q2: The SBOM job downloads the exact `image.tar` artifact that `build-scan` already
scanned and passed, instead of rebuilding the image from source. Why does that
matter for supply-chain integrity?
A (Adam): it double-verifies the integrity of the image, in case build-scan missed
something.
Corrected: it's not about redundant scanning — it's about **identity/provenance**.
If `sbom` ran `docker build` again itself, you could get a different image (code
moved on `main` between jobs, non-reproducible layers, a re-pulled base image with a
new digest). You'd end up with an SBOM describing an image that was never actually
scanned, and possibly a pushed image that doesn't match either. Reusing the same
artifact guarantees: scanned image == SBOM'd image == pushed image.

Q3: Actions are pinned by commit SHA (`@a9c7b0f06e... # v0.36.0`), not by tag
(`@v0.36.0`). What attack does SHA-pinning defend against that tag-pinning does not?
A (Adam): did not recall.
Corrected: a tag is a mutable pointer, like a branch — the repo owner (or anyone who
compromises their account/token) can force-move it to point at different, malicious
code at any time, and your workflow picks up that new code on the next run with
*zero diff* in your own `ci.yml` to catch in review. A commit SHA is a hash of the
actual content — it cannot be made to mean different code without an actual hash
collision. The `# v0.36.0` comment is for humans only; it doesn't affect resolution.
Real-world precedent: the 2025 `tj-actions/changed-files` incident, where a tag was
repointed to inject a secrets-dumping payload into thousands of pipelines.
