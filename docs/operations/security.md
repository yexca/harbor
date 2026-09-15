# Deployment Security

The collection is public to anyone who can reach Harbor, including services
hidden from home. Configure reverse-proxy authentication, a private network, or
a VPN if addresses and icons must be private.

Set `HARBOR_ADMIN_PASSWORD` to protect adding, editing, deleting, home selection,
and icon fetching. An empty value grants editing access to every visitor. The
password is a single shared editing secret, not a user-account system.

Use HTTPS outside a trusted local network. A reverse proxy should serve Harbor
at its host root and preserve the original Host header. Login through the normal
browser flow allows the server to mark its session cookie Secure behind an HTTPS
proxy. Do not expose a second direct HTTP entry point unintentionally.

Icon fetching runs from the NAS and intentionally supports private LAN addresses.
It is available to anyone with editing access. Keep that permission restricted
where the server can reach private systems. Requests do not forward browser
authentication; TLS verification remains enabled.

The provided container runs unprivileged. Keep `/data` writable, the root
filesystem read-only, and unnecessary host mounts or Docker sockets unmounted.
Keep passwords, collection backups, and logs private. They can contain service
addresses and filesystem information.

See [privacy](../../PRIVACY.md), [security reporting](../../SECURITY.md), and
[implementation boundaries](../development/security.md).
