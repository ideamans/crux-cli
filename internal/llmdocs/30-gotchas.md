# Gotchas and limits

- **`-d all` means different things in different subcommands.**
  `crux device -d all` returns one row **per device**.
  `crux history -d all` / `crux record -d all` return **one aggregated record**
  with no per-device split. Reading one as the other silently changes the
  meaning of every number.
- `crux device` accepts at most **50 origins** per invocation.
- `crux history --periods` accepts 1–40 (default 25).
- **CrUX API rate limit is roughly 150 queries per minute per key.** Each origin
  or URL is a separate request, so querying N targets makes N requests. Batch
  work accordingly rather than firing everything at once.
- If the API has no data for a target, it is **skipped with a notice, not an
  error**. Check that every target you asked for came back before drawing
  conclusions.
- `cls` is a unitless ratio; every other metric is milliseconds.
- BigQuery queries cost money against the configured project. The local cache
  exists to avoid repeat charges — `--no-cache` bypasses it.

## Examples

```bash
crux device  -o https://example.com -d phone --months 6 -f json
crux history -o https://web.dev --periods 40 -f json
crux history -u https://web.dev/learn --metrics lcp,cls,inp,fcp,ttfb,rtt -f json
crux record  -o https://web.dev -d desktop -f json
```
