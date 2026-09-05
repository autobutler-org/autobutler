# Quark Docs

Quark is a self-hosted personal cloud that runs on a device in your home. This is the technical documentation
for contributors and people running their own instance.

Looking for setup instructions? Start with the [README](../README.md).

---

## Development

- [Dev Onboarding](./dev-onboarding.md) — Get the codebase running locally
- [UI Components](./ui/index.md) — How to edit the frontend
- [iOS Dev Setup](./ios-dev/README.md) — running the app on your own iPhone
- [iOS App Store Release](./ios-release.md) — building and publishing to the App Store
- [Android Play Store Release](./android-release.md) — building and publishing to Google Play

## Features

- [Authentication](./auth.md) — local auth setup, login, recovery
- [Mobile Setup](./mobile-setup.md) — running the app on a physical Android or iOS device

## Reference

- [API (Swagger)](./swagger/index.html) — Auto-generated API docs (start the backend first)
- [ePub Viewer](./epub/index.md) — Notes on the epub.js integration

## Architecture

- [Codebase Organization Review](./architecture/codebase-organization-review.md) — a one-time audit
  for "AI drift": renames that never finished, safety checks reimplemented instead of reused,
  duplicated UI states, and other places where individually-fine PRs added up to something messier
  than any one of them
