# Security Policy

Harbor is a personal NAS start page. The current development branch receives
fixes; no tagged release has been published by this repository initialization.

Viewing the collection is public to anyone who can reach the server. The optional
administrator password protects modifications and icon requests, not reading
service addresses. Home visibility is not an authorization boundary.

See [deployment security](docs/operations/security.md),
[secure development](docs/development/security.md), and [PRIVACY.md](PRIVACY.md)
for the implemented boundaries and data flows.

## Reporting

This local repository has no configured public reporting endpoint. When a remote
repository is established, maintainers should configure a private reporting
channel and update this page before public distribution. Do not assume an email
address found in Git history is a security contact.

Report an undisclosed vulnerability through an established private maintainer
channel, not a public issue. Include the affected commit, prerequisites,
synthetic reproduction steps, expected/actual behavior, and impact. Do not attach
passwords, session cookies, real service configurations, or unredacted logs.

Authentication bypasses, unsafe icon processing, unauthorized network access,
script injection, and data corruption are relevant reports. Test only systems
you own or have permission to assess.
