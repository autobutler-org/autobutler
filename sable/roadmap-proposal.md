# AutoButler Roadmap Proposal

> Drafted by Sable, 2026-03-16. For discussion with Brandon + James before executing.
> Nothing has been changed in GitHub yet.

---

## Proposed Milestones

| Milestone | Theme | Goal |
|-----------|-------|------|
| **v0.1 — Foundation** | Security, stability, mobile | Ship a product that's secure and usable |
| **v0.2 — Content** | Books, photos, albums | Make the butler worth filling up |
| **v0.3 — Storage & Backup** | Drives, backup, networking | Make it trustworthy long-term |
| **Someday** | Big bets | Needs the core to mature first |

> The existing **v1.0** milestone only has one ticket (#546 Cirrus previews).
> Suggest repurposing it as a "public launch" milestone once the others are done.

---

## New Epics to Create

- **[EPIC] Books / Document Viewer** — PDF/epub reader (Brandon's priority for v0.2)
- **[EPIC] Settings & Device Management** — home for scattered settings tickets
- **[EPIC] Cirrus Polish** — dark mode bug, upload UX, search, tree view, file tagging

---

## Issue → Milestone Mapping

### v0.1 — Foundation
- #507 Security: Basic auth *(epic)*
- #414 Prevent neighbor's butler access *(security bug)*
- #605 Cirrus toolbar dark mode bug
- #564 Custom HTTP errors *(PR #638 open)*
- #426 Auto update toggle *(PR #639 open)*
- #382 New device storage operational on plug-in
- #169 Docs: How to turn butler off
- #168 Docs: How to get help

### v0.2 — Content
- [NEW] Books / Document Viewer epic (Brandon's priority)
- #329 "View in ___" (open file in books/photos/music)
- #440 Recently Uploaded *(epic)*
- #503 Photo Albums *(epic)*
- #399 Favorites and Albums — API layer
- #400 Favorites and Albums — UI
- #540 Photos viewer navigation
- #379 Origin-specific photo library view
- #290 Mobile: Auto-sync iOS photos into Cirrus
- #546 Previews for Cirrus *(currently in v1.0, move here)*

### v0.3 — Storage & Backup
- #517 Backups *(epic)*
- #519 Multiple backup devices *(possible duplicate of #185 — see below)*
- #513 Rename storage devices
- #498 External TCP tunnel / Tailscale *(epic)*
- #417 Mount autobutler as SMB network drive
- #416 Cloud drive location in Finder filepath
- #415 Save security system videos to the butler

### Settings & Device Management *(new epic, spread across v0.1–v0.2)*
- #442 Light mode toggle in settings
- #403 Users can choose color schemes
- #424 View SBOM on settings page
- #348 Show number of devices connected
- #350 Access management in settings
- #421 Metrics: show restarts/power context
- #408 Metrics/Health — Pi health

### Cirrus Polish *(new epic, v0.2)*
- #332 Search feature
- #337 Add tags for files
- #328 Tree view for file explorer
- #260 Create new docx from Cirrus UI
- #273 Multi-upload display improvements
- #256 Make uploader bar cute
- #302 Delete in Cirrus deletes on devices
- #578 Caching file structure in localStorage

### Someday
- #194 Google Docs replacement
- #195 Google Sheets replacement
- #196 Powerpoint viewer
- #198 Google Forms replacement
- #190 Google Meet replacement
- #189 Democratic YouTube replacement
- #193 Investigate mesh compute
- #181 Email server
- #188 Plugin system
- #612 Implement performance budgets
- #610 Optimize images for web performance

> **Note on Office Suite:** #194, #195, #196, #198 should probably be children of a single
> **[EPIC] Office Suite** rather than three separate epics. Suggest consolidating.

---

## Duplicates to Resolve

| Issues | Problem | Suggestion |
|--------|---------|------------|
| #185 vs #519 | Both are "backup to external drive" | Close one, link to the other |
| #416 vs #417 | SMB mount vs Finder path — very similar | Keep both but put under same epic |
| #194/#195/#196 | Three separate "Google replacement" epics | Merge into one [EPIC] Office Suite |

---

## Tickets Needing a Decision

- **#441 Unified View Toggle** — Cirrus Polish or Unified File Systems epic?
- **#375 Calendar week view** — Is Calendar actively being built? No epic for it.
- **#368 Calendar reminders** — Same question.
- **#385 Zero data defaults updates** — Unclear scope, needs review.
- **#87 Login and customer dashboard** — Old ticket, possibly closeable?
- **#85 Set up a PiHole** — Old, closeable?
- **#84 Set up a NAS** — Old, closeable?

---

## Questions for James

1. Mobile app status — what's the current state, what's left for v0.1?
2. Calendar — is this on the roadmap or parked?
3. Duplicates #185/#519 — which one to keep?
4. Office Suite — are Docs/Sheets/Meet replacements realistic near-term or truly someday?
5. Old tickets #84/#85/#87 — close or keep?
