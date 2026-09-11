# Running release-manager as multiple replicas

**Status:** Analysis — not yet implemented
**Date:** 2026-06-09
**Question:** What does it take to run multiple replicas of the release-manager so it answers correctly on its APIs and handles its flows?

---

## Goals

1. **Run N replicas of the release-manager safely.** The fleet must tolerate losing any
   single replica (rolling deploys, node failure, OOM) without losing correctness or
   availability of either reads or flows.
2. **Every read endpoint returns fresh data from every replica.** A client hitting
   `/status`, `/describe/*`, or `/policies` must get the same, up-to-date answer no
   matter which replica the load balancer routes it to — no stale-read window that
   depends on which pod received the last git webhook.
3. **Achieve freshness via near-instant fanout propagation.** On every config-repo push,
   *all* replicas sync their local clone, driven by a fanout broadcast of the
   config-changed notification — not a per-request git fetch (too slow on the read path)
   and not only a coarse background ticker (too stale).
4. **Preserve serialized, HA write flows.** Releases and artifact processing stay
   effectively serialized fleet-wide with automatic failover, with no change to the
   existing quorum + single-active-consumer queue.
5. **No new stateful infrastructure.** Multi-replica support must work with a plain
   `Deployment` and per-pod ephemeral storage — no StatefulSet, no shared volume, no
   database.

### Non-goals

- **Parallel write throughput.** The service pushes to a single `master` branch; N
  replicas buy HA failover, not concurrent writes. Removing single-active-consumer is
  explicitly out of scope.
- **Strong read-after-write consistency on a per-request basis.** We accept a small,
  bounded propagation delay (sub-second to low seconds) between a push and all replicas
  reflecting it, rather than syncing synchronously on every read.

### Success criteria

- [ ] With `replicas: N` (N ≥ 3) behind the load balancer, the same read request issued
      repeatedly returns identical, current data regardless of which replica serves it.
- [ ] After a config-repo push, **all** replicas reflect the change within a bounded
      target (e.g. **< 2s** p99), measured end-to-end from webhook receipt to a fresh
      read on the replica that did *not* receive the webhook.
- [ ] Killing the replica that currently holds the active flow consumer causes another
      replica to take over and continue processing releases/artifacts with no manual
      intervention and no duplicate pushes.
- [ ] Read-endpoint p99 latency does **not** regress versus single-replica today (i.e.
      freshness is not bought with a per-request git fetch).
- [ ] A newly started / rejoining replica converges to fresh data without operator
      action (initial clone on boot, then fanout updates).

---

## TL;DR

- **Flows (releases, artifacts) are already multi-replica safe.** The AMQP queue is a
  quorum queue declared with `x-single-active-consumer: true`, so across the whole
  fleet only one replica processes at a time, with automatic failover. Running N
  replicas gives **HA failover, not write throughput** — which is correct for a
  service that pushes to a single git branch. **Do not remove single-active-consumer.**
- **Reads are the blocker.** The read APIs serve from a per-replica local clone of the
  config repo that is refreshed **only** by the inbound GitHub push webhook. A load
  balancer delivers each webhook to exactly one replica, so the other replicas serve
  **stale** data indefinitely.
- **The one real fix:** make every replica keep its own clone fresh by **fanout-broadcasting
  config-changed notifications** so all replicas sync immediately on push (see Goals). A
  background ticker is a simpler fallback but only bounds staleness coarsely. Deployment-wise
  a plain `Deployment` with `emptyDir` is sufficient — no StatefulSet, no shared volume.

---

## Architecture context

The release-manager has **no database**. Its sources of truth are:

- the **config git repo** (remote) — releases, policies, status
- **S3** — artifacts

On startup each replica clones the config repo into a **local temp dir**
(`internal/git/git.go:55` `InitMasterRepo`), guarded by a per-process
`sync.RWMutex` (`internal/git/git.go:46`). Every API read and every write copies
*that local clone* (`copyMaster`, `git.go:134`) and operates on the copy.

**This single fact drives everything:** the local clone is per-replica state, and
correctness depends entirely on keeping each replica's clone fresh.

---

## Writes / flows — already safe across replicas

The AMQP queue is declared as a **quorum queue with `x-single-active-consumer: true`**:

```go
// internal/amqp/consumer.go:121
queueArgs := amqp.Table{
    "x-queue-type":             "quorum",
    "x-single-active-consumer": true,
}
```

Across *all* connected replicas, RabbitMQ allows only **one** consumer to process at a
time, failing over to another consumer if the active one dies. Combined with the
git-push-conflict retry (`internal/flow/flow.go:124` → `SyncMaster` on
`ErrBranchBehindOrigin`), release and artifact flows are effectively serialized
fleet-wide.

**Implication:** N replicas buy you HA failover, not parallel write throughput. That is
the correct design — the service pushes to a single `master` branch on the config repo.

> ⚠️ **Do not** remove `x-single-active-consumer` to "enable scaling." It would let
> multiple replicas push to the config repo concurrently, increasing push contention
> for no benefit.

---

## Reads — the actual blocker

The read APIs (`/status`, `/describe/*`, `/policies`) **do not sync before reading**.
`SyncMaster` is only called from two places:

- the GitHub push webhook handler (`cmd/server/http/github_webhook.go:36`)
- the write-conflict retry path (`internal/flow/flow.go:130`)

Reads simply copy the local master clone and walk the git log. The local clone is
refreshed **only** when this replica receives a GitHub push webhook.

With a load balancer in front of N replicas, GitHub delivers each push webhook to
**exactly one** replica:

```
Replica A receives webhook → SyncMaster → serves fresh data
Replicas B, C            → never sync   → serve STALE releases/policies indefinitely
```

This is why a naive `replicas: 3` makes the API return different/wrong answers
depending on which pod the request lands on.

---

## What it takes to run multiple replicas correctly

1. **Keep every replica's clone fresh** (the core fix). **Chosen approach: fanout
   broadcast.**
   - **Fanout broadcast (chosen)** of config-changed notifications via a dedicated
     exchange with a **per-replica exclusive, auto-delete queue**, so all replicas sync
     immediately on push and meet the `< 2s` freshness target. The current flow queue
     (`internal/amqp/consumer.go:120`) is a single named durable queue with
     `x-single-active-consumer` — by design only *one* replica consumes each message, so
     it **cannot** be reused for broadcast. This requires a new exchange/queue topology:
     each replica declares its own exclusive queue (`exclusive: true`, auto-deleted on
     disconnect, no single-active-consumer) bound to the notification exchange, and on
     delivery calls `SyncMaster`.
   - **Background sync ticker (fallback)** per replica calling `SyncMaster` every N
     seconds. No new topology, but only bounds staleness coarsely and won't hit the
     sub-2s target — keep as a safety net / fallback, not the primary mechanism.

2. **Keep the AMQP flow queue exactly as-is** (quorum + single-active-consumer). Flows
   are already correct; you get failover for free.

3. **Verify SQS artifact processing** is a competing-consumer setup (SQS is by nature) —
   fine across replicas, no change expected.

4. **Deployment.** The master clone lives in an **ephemeral per-pod temp dir**, so a
   plain `Deployment` with `emptyDir` works — no StatefulSet, no shared ReadWriteOnce
   PV. `/ping` liveness/readiness probes are fine.
   - The `hostPath` volumes flagged during analysis are in the **e2e-test manifest only**
     (`e2e-test/release-manager.yaml`, local git-server fixture), **not** a production
     blocker. Confirm against the actual prod Helm chart (likely in a separate deploy
     repo) before changing replica count.

---

## Design

This is the implementation design for the chosen fanout-broadcast approach. It resolves
the topology, payload, wiring, and failure-handling questions left open by the analysis.

### Note on the actual broker layering

The production AMQP path is **not** `internal/amqp` directly — it is the
`broker.Broker` interface (`internal/broker/broker.go:9`) implemented by
`internal/broker/amqpextra`, which *wraps* `internal/amqp.Worker`. The amqpextra broker
declares a **single** durable quorum queue with `x-single-active-consumer`, bound to one
topic exchange with routing key `#` — all event types are multiplexed through it and
demuxed by message `Type` (`internal/broker/amqpextra/consumer.go:25`). The in-memory
broker (`internal/broker/memory`) is the second implementation, used for local/dev and
tests. Any broadcast capability must be added at the `broker.Broker` level so both
implementations stay in sync.

### Decisions

1. **Extend `broker.Broker` with broadcast methods.** Add `PublishBroadcast(ctx,
   Publishable)` and `StartBroadcastConsumer(handler func([]byte) error)`. The
   `amqpextra` implementation backs them with a **dedicated fanout exchange** and a
   **server-named, exclusive, auto-delete queue** (one per replica, so every replica
   receives every config-changed message). The `memory` implementation does an
   in-process fan-out so single-process tests/dev keep working.

2. **SHA-carrying payload with skip-if-current.** A new `ConfigChangedEvent{SHA string}`
   implements the existing `broker.Publishable` interface (`Type`/`Marshal`/`Unmarshal`).
   The broadcast handler compares the SHA to the replica's local master HEAD (new
   `git.Service.MasterHash()` getter, read under the existing `RLock`) and **skips** the
   `fetch`+`pull` when already current. This no-ops the publisher's own self-delivered
   message and coalesces rapid/duplicate pushes for free.

3. **Publish from the webhook via an injected hook.** A `func(context.Context, sha
   string) error` hook is added to `http.NewServer` (`cmd/server/http/http.go:32`) and
   threaded into the webhook handler (`cmd/server/http/github_webhook.go:36`), wired in
   `cmd/server/command/start.go` to `brokerImpl.PublishBroadcast`. The handler calls
   `SyncMaster`, reads `MasterHash`, then publishes. This matches the existing injected
   publish-hook idiom on `flow.Service` (`PublishReleaseArtifactID`/`PublishNewArtifact`,
   `start.go:511`). Broadcast-publish failure is **log-only** — the webhook still returns
   `200`.

4. **Single-worker broadcast consumer, ack-always.** The per-replica queue is consumed
   by a single worker so deliveries serialize; with skip-if-current a redundant delivery
   costs only a HEAD read, not a write-locked pull. The handler **acks unconditionally**
   and logs on `SyncMaster` failure — no nack/requeue (same data would loop on an
   exclusive queue), relying on the next broadcast or the ticker to recover.

5. **Coarse ticker backstop.** A configurable (~60s) background `SyncMaster` ticker per
   replica heals missed broadcasts (reconnects, and the brief cold-start window before a
   replica's queue is bound). Broadcasts deliver the `< 2s` freshness; the ticker is only
   a safety net.

### New AMQP primitives

`internal/amqp` currently hardcodes quorum + single-active-consumer + `exclusive: false`
on every declared queue and `topic` on every declared exchange. The broadcast path needs:

- `ConsumerConfig` gains optional fields for **exclusive / auto-delete / server-named
  queue** and **omitting single-active-consumer**, plus a **fanout** binding.
- The publisher (`internal/amqp/publisher.go:96` `declareExchange`) gains support for
  declaring a `fanout` exchange (currently hardcoded `topic`).

The existing flow-queue declaration (`internal/amqp/consumer.go:121`) stays
**byte-for-byte unchanged**.

### Lifecycle

`StartBroadcastConsumer` and the ticker each run in their own cancellable goroutine in
`start.go`, alongside the existing `StartConsumer`. The exclusive auto-delete queue is
removed automatically by RabbitMQ when the replica's connection closes.

### Verification against success criteria

- **Fresh on every replica / `< 2s` after push** — broadcast → skip-if-current →
  `SyncMaster` on all replicas.
- **No read-latency regression** — single-worker + skip-if-current avoids extra
  write-locked pulls; the write lock is held only for genuine changes, as today.
- **Failover / convergence** — flow queue unchanged (HA); ticker + initial clone
  converge new/rejoining replicas with no operator action.

---

## Bottom line

Flows already work across replicas. The one thing standing between you and correct
multi-replica behavior is **stale reads on replicas that did not receive the git
webhook**. Fix master-sync propagation via **fanout broadcast** (with an optional ticker
as a safety net) and you're there.

## Open follow-ups

- [ ] Confirm the production deployment manifest in the deploy/Helm repo (volumes,
      current replica count, probes).
- [x] ~~Decide between background sync ticker vs. fanout-broadcast for clone freshness.~~
      **Decided: fanout broadcast** for near-instant freshness (< 2s target); ticker
      retained only as a backstop. See **Design**.
- [x] ~~Design the fanout exchange + per-replica exclusive-queue topology and the
      config-changed notification payload.~~ See **Design** (SHA-carrying
      `ConfigChangedEvent`, fanout exchange + server-named exclusive auto-delete queue).
- [ ] Confirm SQS artifact consumer is competing-consumer (expected, but verify).

## Key references

| Concern | Location |
|---|---|
| Master clone init | `internal/git/git.go:56` (`InitMasterRepo`) |
| Clone copy for reads/writes | `internal/git/git.go:165` (`copyMaster`) |
| Sync master (write lock) | `internal/git/git.go:105` (`SyncMaster`) |
| Broker interface (Publish/StartConsumer) | `internal/broker/broker.go:9` |
| Production broker (wraps internal/amqp) | `internal/broker/amqpextra/consumer.go:25` |
| Quorum + single-active-consumer queue | `internal/amqp/consumer.go:121` |
| Exchange declare (hardcoded topic) | `internal/amqp/publisher.go:96` (`declareExchange`) |
| HTTP server / webhook wiring | `cmd/server/http/http.go:32` (`NewServer`), `:81` |
| Webhook-triggered sync | `cmd/server/http/github_webhook.go:36` |
| Write-conflict retry → sync | `internal/flow/flow.go:131` |
| Broker construction / event hooks | `cmd/server/command/start.go:511`, `:615` |
