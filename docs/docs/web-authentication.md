---
title: Web authentication
---

Authentication is disabled by default, preserving existing deployments. Enable it
for the web server with `postie-web --auth` or `POSTIE_AUTH_ENABLED=true`, then
restart the server. `--auth=false` overrides the environment setting. These
web-only startup options do not change the shared desktop/CLI configuration.
Invalid environment values fail startup. On the first authenticated visit, create
an administrator account before completing NNTP setup. Registration closes after
that account is created. Usernames need at least three characters; passwords
must be 12–72 bytes. You can sign in with the username or optional email address.

Authentication adapts AltMount's direct login mechanism: bcrypt password hashing
and signed JWT cookies using `go-pkgz/auth`. Credentials, signing keys and
revocable sessions persist in Postie's SQLite database. Sessions last 24 hours.
Signing out revokes the current session and closes its WebSocket connections;
expired sessions also close their connections. Each browser login is a separate
session. Desktop Wails and CLI operation continue without a web login.

When enabled, web pages, application APIs, logs, filesystem browsing and API-key management
require a valid session. External `/api/v1/queue/upload` clients retain their
existing API key authentication and API-enabled setting. Arr webhook receivers
retain their configured query API key; other Arr APIs require a browser session.
`/live` remains public for health checks.

For HTTPS behind a reverse proxy, preserve the original `Host` header and set
`POSTIE_COOKIE_SECURE=true` on the web server. This marks the cookie Secure and
requires HTTPS browser origins, even when the proxy connects to Postie over
HTTP. Direct TLS requests automatically use Secure cookies. Cookies are
HttpOnly, scoped to the host, and SameSite Strict. Avoid exposing a fresh
installation to other users before creating its first administrator account.

This adaptation supports one administrator. It does not share credentials or
sessions with AltMount, and does not include account editing or renewal UI.
After session expiration, sign in again. Login and registration are limited to
10 attempts per minute per direct peer IP; clients behind a reverse proxy share
that limit. Requests from browser origins must match the original Host and
protocol. Forwarded headers are deliberately not trusted.

Disabling authentication hides login/logout controls and keeps existing UI,
API and WebSocket behavior. Accounts and sessions remain in the application
SQLite database, so re-enabling protection and server restarts preserve them.
There is no password-reset UI in this initial implementation; back up the
application database and keep the administrator credentials securely.

The direct-login adaptation was checked against AltMount revision
[`ac62dce`](https://github.com/kipsilabs/altmount/tree/ac62dce42ed779cefbadbf23dc4973f4c45de470/internal/auth)
and its [HTTP authentication handlers](https://github.com/kipsilabs/altmount/blob/ac62dce42ed779cefbadbf23dc4973f4c45de470/internal/api/auth_handlers.go).
Postie reuses its bcrypt and `go-pkgz/auth` JWT approach, with a singleton
administrator and persisted session revocation suited to Postie's web mode.
