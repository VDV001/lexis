# Test-only age keypair

**These keys exist solely for smoke / unit tests.** They are committed to the
repository on purpose so the test harness has a deterministic recipient + identity
pair without external setup.

| File | Content |
|---|---|
| `test_identity.txt` | private key (full age identity file) |
| `test_pubkey.txt`   | public key (single `age1…` line) |

**Never reuse these keys for production backups.** The production keypair is
generated separately (zone of the project owner, kept in 1Password + local
Keychain) and the public key is injected into the backup container via the
`AGE_PUBLIC_KEY` environment variable.

To regenerate the test keys (e.g. on key-rotation drills):

```sh
age-keygen -o test_identity.txt
age-keygen -y test_identity.txt > test_pubkey.txt
```
