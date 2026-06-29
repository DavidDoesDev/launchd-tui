# Feature backlog

Ideas not yet committed to. Promote to a GitHub issue when ready to build.

## Distinctive / launchd-specific
- **Per-agent mini-sparkline** — a tiny activity spark on each list card's status line, so the sidebar shows which agents are busy at a glance.
- **Desktop notification on failure** — fire a macOS notification when an agent flips to errored / non-zero exit. Turns the dashboard into a watchdog.

## Core usefulness
- **Filter / search** (`/`) — filter the agent list and/or search within the log viewport.
- **`kickstart` fallback** — modern `launchctl kickstart gui/$(id -u)/<label>` for agents that ignore `start` on Ventura+.
- **Open plist / log in `$EDITOR`, reveal in Finder** — quick jumps out to edit or inspect.
- **Resource usage** — CPU/mem for running PIDs via `ps`, shown in the Info tab.

## Polish
- **Sort / group** — by status, name, or a `category` tag in config.
- **Confirm-before-stop/restart** — safety toggle (was speced into the settings menu, never built).
- **More themes** — beyond mocha/latte/gruvbox/nord.

## Distribution
- **Homebrew** — GoReleaser + tap so it's `brew install`-able.

---

## Done
- **Schedule awareness** — cadence + live next-run estimate in the Info tab.
