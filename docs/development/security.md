# Secure Development

## Requests and Sessions

Reuse the editor middleware for mutations and icon fetching. Public service
reads are intentional. Preserve the custom same-origin request header, Origin
and Fetch Metadata checks, JSON body limits, and rejection of unknown fields.

Session tokens are random, server-side, memory-only, and expire after 24 hours.
Cookies use HttpOnly and SameSite=Strict; HTTPS browser login also sets Secure.
Keep authentication errors generic and avoid exposing upstream responses, private
paths, credentials, or cookies in API errors.

## Outbound Icons

Use `newIconClient` and its bounded transport for every icon request. It permits
private global-unicast LAN addresses intentionally but rejects loopback,
link-local, multicast, unspecified, and scoped addresses. Resolve DNS, validate
the returned addresses, and dial the validated IP rather than resolving again.

Only HTTP(S) URLs without embedded credentials are accepted. Redirects are
bounded and validated; declared icons and redirects may use another origin under
the same address policy. This is not a same-origin-only fetch policy. Browser
cookies and service credentials are not forwarded, and TLS verification stays on.

The current discovery budget is eight seconds, with a four-second HTTP client
timeout, bounded reads (1 MiB page / 512 KiB icon), and at most three simultaneous
discovery requests. Preserve cancellation and these bounds when extending it.

Validate both uploaded and downloaded icons. SVG is restricted to an inert
subset, excluding scripts, external resources, embedded HTML, and directives.
Browser rendering uses local data URIs and textContent for service-provided text.

## Persistence and Development Data

Keep temporary writes in the target data directory and publish only after the
write succeeds. Never overwrite invalid startup data or silently reset a store.
Tests must use temporary directories and synthetic endpoints, not a real NAS
configuration. Protect compatibility and failure behavior with focused tests.

Run `make sensitive-check` before commits. Do not add inline suppression comments
or broad URL allowlists. A public reference must have an exact file-scoped entry
and a reason in [privacy-allowlist.json](../../scripts/privacy-allowlist.json).
Review staged content independently of the scanner.

See [deployment security](../operations/security.md), [privacy](../../PRIVACY.md),
and [reporting](../../SECURITY.md).
