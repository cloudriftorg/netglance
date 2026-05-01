# Security policy

## Supported versions

Until we hit `v1.0.0`, only the latest tagged release is supported with
security fixes. Older tags will not get backports.

| Version | Supported |
|---|:---:|
| latest tag | ✅ |
| anything older | ❌ |

## Reporting a vulnerability

**Please do not open a public GitHub issue for security problems.**

Instead, send a private report via GitHub's security advisory flow:

➡ <https://github.com/netglance/netglance/security/advisories/new>

Include:
- a clear description of the issue and its impact,
- steps to reproduce (or a proof-of-concept),
- the netglance version you tested,
- whether you'd like to be credited in the advisory (and how — handle, name,
  link).

You'll get an acknowledgement within ~5 working days. Triage, fix and
coordinated disclosure timing depend on severity; for high-impact issues
we'll usually publish the advisory together with a patched release.

If GitHub Security Advisories aren't an option for you, email the
maintainer (see the email in the project's `Makefile` `MAINTAINER=` field
or the latest commit's author).

## Out of scope

The following are not considered vulnerabilities:

- Issues requiring physical access or local root to the host running
  netglance.
- Issues only reproducible in unsupported install configurations
  (e.g. running netglance unprivileged without `arp-scan`'s setuid bit).
- Self-XSS or vulnerabilities in third-party browsers, networks or
  hosting platforms.
- Missing security headers without a concrete exploit. (We do welcome
  hardening PRs — open them publicly.)

Thanks for keeping the project safer.
