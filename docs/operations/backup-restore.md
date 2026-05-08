# Backup & Restore Runbook

> **Purpose:** recover Lexis production data from an encrypted off-site
> backup. Read this before you run anything destructive.
>
> **Audience:** project owner / on-call engineer who has access to the
> production age private key.
>
> **Time budget for end-to-end recovery:** ≈ 30–60 minutes after you
> decide to restore.

---

## What gets backed up

PostgreSQL of the Lexis production deployment, daily at 03:00 UTC, via
the `backup` service in `docker-compose.yml` under the `backup` profile.

| Stage | Tool | Output |
|---|---|---|
| Dump | `pg_dump --format=plain --no-owner --no-privileges` | `lexis-<UTC ISO>.sql` |
| Encrypt | `age -r <AGE_PUBLIC_KEY>` | `lexis-<UTC ISO>.sql.age` |
| Upload | `aws s3 cp` (Selectel ru-1, S3-compatible) | `s3://<bucket>/<file>` |
| Retain | `retention.sh` (7 daily + 4 weekly + 3 monthly) | older objects deleted |

The S3 bucket lives **on a different provider from the host running
PostgreSQL** (Rule 4 of `reflective-agent-defaults` v1.3 — backups must
be outside the blast radius of the source). A second retention layer is
configured as an S3 lifecycle policy on the bucket, so even if the cron
container fails, objects older than ~96 days fall off automatically.

## What is NOT backed up

- **Redis** — only ephemeral cache, regenerated from Postgres + user actions.
- **Plaintext SQL on the backup container's disk.** `backup.sh` deletes
  the unencrypted `.sql` file as soon as the `.age` artefact is written;
  the bucket only ever sees encrypted bytes.
- **The age private key.** It lives in the project owner's 1Password
  vault (primary) plus a local-Mac Keychain copy (recovery). It is
  never in CI, never on the VPS.

---

## One-time setup

These steps are done once when standing up the backup pipeline. They
are listed for documentation; they are NOT part of the daily flow.

1. **Generate the production age keypair** on a trusted local machine:

   ```sh
   age-keygen -o ~/lexis-prod-age-identity.txt
   age-keygen -y ~/lexis-prod-age-identity.txt > ~/lexis-prod-age-pubkey.txt
   ```

2. **Move the private key to 1Password** (primary). Add a Recovery
   item type with the contents of `lexis-prod-age-identity.txt` in a
   secure-note field. Add a second copy to the local Mac Keychain as a
   secure-note (recovery if 1Password is unreachable).

3. **Wipe the working file** (`shred -u` or equivalent) once the keys
   are stored in 1Password and Keychain.

4. **Create the Selectel S3 bucket** (e.g. `lexis-postgres-backups`,
   region `ru-1`) and a dedicated IAM user with `PutObject` + `GetObject`
   + `DeleteObject` + `ListBucket` scoped to that bucket only.

5. **Configure the lifecycle policy** on the bucket: expire objects
   after 96 days. This is the safety net if the cron-side retention
   fails to run.

6. **Provision env vars** on the host running the cron container:

   | Variable | Value |
   |---|---|
   | `DATABASE_URL` | production Postgres connstring |
   | `S3_ENDPOINT_URL` | `https://s3.ru-1.storage.selcloud.ru` |
   | `S3_BUCKET` | bucket name from step 4 |
   | `AWS_ACCESS_KEY_ID` | IAM user from step 4 |
   | `AWS_SECRET_ACCESS_KEY` | IAM user from step 4 |
   | `AGE_PUBLIC_KEY` | contents of `lexis-prod-age-pubkey.txt` |

7. **Cron the daily run.** The `backup` service is opt-in (compose
   profile `[backup]`). Schedule it on the host:

   ```cron
   0 3 * * * cd /opt/lexis && docker compose --profile backup run --rm backup
   ```

---

## Daily flow

There is none from your side. The cron task runs `backup.sh` inside the
`backup` container. Logs land in the host's `journald` /
`docker logs`. Failures surface as the cron job's non-zero exit which
your host monitoring should alert on.

To **inspect a recent run** by hand:

```sh
docker compose --profile backup logs --tail 200 backup
```

To **list current bucket contents** (sanity check):

```sh
aws --endpoint-url "$S3_ENDPOINT_URL" s3 ls "s3://$S3_BUCKET/"
```

---

## Manual ad-hoc backup

You normally do not need this. Use it before risky migrations or schema
changes:

```sh
docker compose --profile backup run --rm backup
```

Or for a **dump-only artefact** (no encryption, no upload — useful for
debugging a snapshot locally without involving S3):

```sh
docker compose --profile backup run --rm \
    -e BACKUP_DRY_RUN=1 \
    -v "$(pwd)/local-dumps:/backup/dumps" \
    backup
```

The plaintext dump lands in `./local-dumps/lexis-<UTC ISO>.sql`.

---

## Recovery procedure (the critical path)

### Step 0 — pause before you act

A botched restore makes data loss permanent. Before running anything:

1. **Confirm you actually need to restore.** Has data really been lost,
   or is this a failed read? Is replication catching up?
2. **Take a snapshot of current state** of the live DB *first*, even if
   it looks corrupt. `pg_dump` of the live volume to disk → the
   investigation surface stays intact.
3. **Coordinate.** Tell stakeholders the API will be unavailable for the
   restore window.

### Step 1 — pull the age private key

From 1Password ("Lexis prod age identity"). Save to a temp path on the
local Mac you are operating from:

```sh
op read "op://Lexis/Lexis prod age identity/notes" > /tmp/age-identity.txt
chmod 600 /tmp/age-identity.txt
```

If 1Password is unavailable: pull from local Keychain. Last resort: a
hardware backup of the Mac. **Never** re-create the keypair — older
backups become permanently unrecoverable.

### Step 2 — identify which backup to restore

```sh
docker compose --profile backup run --rm --no-deps \
    -e AWS_ACCESS_KEY_ID=... \
    -e AWS_SECRET_ACCESS_KEY=... \
    --entrypoint sh backup -c \
    "aws --endpoint-url $S3_ENDPOINT_URL s3 ls s3://$S3_BUCKET/" | \
    sort
```

Choose the latest dump that is **prior to** the corruption you are
recovering from. Note the exact object name, e.g.
`lexis-20260507T030000Z.sql.age`. You will type this back as
confirmation in step 4.

### Step 3 — prepare the target database

Decide whether to restore into:

- **The existing production DB.** Drop and recreate it first:

  ```sh
  psql "$ADMIN_DB_URL" -c "DROP DATABASE langtutor;"
  psql "$ADMIN_DB_URL" -c "CREATE DATABASE langtutor OWNER langtutor;"
  ```

- **A fresh staging DB**, restore there first, validate, then swap. This
  is the safer option and is **strongly recommended** unless you have
  already paused the application.

### Step 4 — run the restore

```sh
echo "lexis-20260507T030000Z.sql.age" | \
    docker compose --profile backup run --rm --no-deps \
        -e S3_ENDPOINT_URL="$S3_ENDPOINT_URL" \
        -e S3_BUCKET="$S3_BUCKET" \
        -e AWS_ACCESS_KEY_ID="$AWS_ACCESS_KEY_ID" \
        -e AWS_SECRET_ACCESS_KEY="$AWS_SECRET_ACCESS_KEY" \
        -e AGE_IDENTITY_FILE=/keys/age-identity.txt \
        -v /tmp/age-identity.txt:/keys/age-identity.txt:ro \
        --entrypoint /backup/restore.sh backup \
        "lexis-20260507T030000Z.sql.age" \
        "$TARGET_DATABASE_URL"
```

`restore.sh` will:

1. Print the critical-action block.
2. Read your typed confirmation (the `echo` you piped in).
3. Reject and exit non-zero if it does not match exactly. **A typo
   means you have to start over** — that is the design, not a bug.
4. On match: download → `age -d` → `psql -v ON_ERROR_STOP=1`.

### Step 5 — verify

```sh
psql "$TARGET_DATABASE_URL" -c "\dt"             # tables present?
psql "$TARGET_DATABASE_URL" -c "SELECT count(*) FROM users;"
psql "$TARGET_DATABASE_URL" -c "SELECT max(created_at) FROM users;"
```

The `max(created_at)` should match (within a few hours) the timestamp
embedded in the backup filename. If it is much earlier — you restored
the wrong dump. Repeat from step 2 with a newer object.

### Step 6 — bring traffic back

If you restored into staging: swap connection strings. If you restored
in place: restart `api`.

```sh
docker compose restart api
```

### Step 7 — clean up

```sh
shred -u /tmp/age-identity.txt
docker compose --profile backup down
```

---

## Smoke test ("can a new operator do this in an hour")

Run the integration smoke against MinIO to convince yourself the
pipeline is healthy without touching production:

```sh
./scripts/test/smoke_backup_restore.sh
```

It exercises the complete loop (backup → S3 → download → decrypt →
psql) end to end, against an isolated `postgres-restore-target` so
your dev DB is untouched. The same script runs every Monday in CI
(`.github/workflows/backup-restore-test.yml`).

---

## Troubleshooting

### `ERROR: backup pipeline cannot start — missing required env vars`

`backup.sh` validates all six required vars at startup. Compare your
host environment against the table in step 6 of the one-time setup.

### `aws: The specified bucket does not exist`

Check `$S3_BUCKET` matches the Selectel-side bucket name exactly (case
sensitive). If you recreated the bucket, the IAM user from setup step 4
needs reattaching.

### `age: failed to decrypt: no identity matched any of the recipients`

You are decrypting with the wrong private key. Make sure
`/tmp/age-identity.txt` is the full identity file (starts with
`AGE-SECRET-KEY-...`) for the public key that was active when the dump
was made.

If the public key was rotated: each old dump can only be decrypted with
the private key that was active at *its* time. This is why we never
rotate the production age keypair without keeping the old private key
indefinitely (it stays in 1Password with a clear "old, do not delete"
note).

### `Confirmation rejected: typed name does not match expected`

You typed the bucket object name with a typo or extra whitespace. The
gate is intentionally strict (Rule 5 v1.3). Run the command again from
the top with a clean copy-paste.

### `psql: ERROR: relation "..." already exists`

You are restoring into a non-empty target. Drop the target DB first
(step 3) or restore into a fresh staging DB.

### CI workflow is failing

Check the latest run on the `Backup Restore Test` workflow page. Most
common: a transient docker pull failure on `minio/minio:latest`. Re-run
the workflow from the GitHub UI.

---

## Quarterly drill (recommended)

Once per quarter, run the full recovery procedure (steps 0–7 above)
into a staging DB. Time it. Note any friction in this runbook and fix
the runbook *before* you forget.

A drill that takes longer than 60 minutes is a sign the runbook is
stale or the pipeline has accreted complexity. Treat it as a real
finding.

---

## Standard cross-references

- **Rule 4 (infrastructure-side enforcement) — `reflective-agent-defaults v1.3`**:
  scoped IAM token + bucket on a separate provider + lifecycle policy
  layer.
- **Rule 5 (out-of-band confirmation) — `reflective-agent-defaults v1.3`**:
  the typing-exact-name gate at restore time.
- **Rule 11.2 (re-injection of rules into immediate context) — `reflective-agent-defaults v1.3`**:
  the critical-action block printed by `restore.sh` before reading
  confirmation.
