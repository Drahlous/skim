# Security Policy

## Supported Versions

skim is a small, actively developed CLI tool. There is no long-term support
branch — only the latest release and the `main` branch receive security
fixes. If you're running an older release, please upgrade to the latest
before reporting an issue, in case it's already fixed.

## Reporting a Vulnerability

Please **do not** open a public GitHub issue for security vulnerabilities.

Instead, report it privately using GitHub's
[private vulnerability reporting](https://github.com/Drahlous/skim/security/advisories/new)
feature (Security tab → **Report a vulnerability**). This opens a private
discussion with the maintainer without disclosing the issue publicly.

Please include as much of the following as you can:

- A description of the vulnerability and its potential impact
- Steps to reproduce, or a proof-of-concept (a crafted `.log` or `.tat`
  file, for example, if that's how it's triggered)
- The version/commit of skim you tested against

### What to expect

This is a solo-maintained project, so response times are best-effort:

- Acknowledgement of your report within a few days
- An initial assessment of severity and next steps once triaged
- Credit in the fix's release notes, if you'd like it (or anonymity, if
  you'd prefer)

Please give a reasonable amount of time to investigate and release a fix
before any public disclosure.

## Scope

skim reads and renders local log files and `.tat` filter files; it does not
make network requests or execute external commands on its own. Reports most
relevant to this project involve things like:

- Crashes, hangs, or memory-safety issues triggered by a malformed log or
  `.tat` file
- Path traversal or unsafe file handling
- Issues in the `$EDITOR` handoff used for in-app filter editing (see
  `ui/filtereditor.go`)

General Go toolchain or third-party dependency vulnerabilities are welcome
too — feel free to report those the same way, or open a public issue if
there's already a public CVE for the dependency.
