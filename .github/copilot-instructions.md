Purpose
-------
These instructions tell GitHub Copilot how to handle programming in this repository.

Key rule (always)
-----------------
- Respect the linting and formatting conventions of the various linting and formatting configurations and tools being used.

Vue SFC layout
--------------
Always order the sections of a Vue Single File Component (SFC) as follows:
1. `<template>` block
2. `<script lang="ts" setup>` block
3. `<style lang="scss">` block, adding `scoped` if styles are component-scoped

No-scroll page layout principle
--------------------------------
- **The page body/viewport should NEVER scroll.** Page scrolling creates a poor user experience.
- Pages should use viewport-constrained layouts (e.g., `height: 100vh`, `overflow: hidden` on main containers).
- When content needs to scroll, it should be within an **explicit, contained scrollable region** (e.g., a table tbody, a content area, a modal body).
- Use flexbox layouts with `flex: 1` and `min-height: 0` to create flexible containers that fit within the viewport.
- Example pattern:
  ```css
  .page-container {
      height: 100vh;
      display: flex;
      flex-direction: column;
      overflow: hidden;
  }
  .scrollable-content {
      flex: 1;
      overflow-y: auto;
      min-height: 0;
  }
  ```

Backend development assumptions
-------------------------------
- Assume the developer is running the backend via `make watch` and that it will auto-reload on code changes.
- Never run the `make generate` target. Just assume the code is generated automatically as a part of `make watch`.
- Never attempt to start, stop, or restart the backend server yourself.
- Focus on code changes only; the running server will pick them up automatically.

Mock UI Elements
-------------------------------
- When generating mock UI elements for testing or demonstration purposes, ensure they are clearly marked as mock components.
- To mark a UI element as mock, use the `mock_badge` component.
- If most of a page is a mock, use the `mock_banner` component.

API endpoint architecture
-------------------------------
- Always separate business logic from HTTP handling in API endpoints.
- API endpoints should only:
  1. Extract data from the request and request URL
  2. Call a service/library function to perform the actual operation
  3. Construct an API response from the result
- Business logic functions must live in `pkg/util/` packages (e.g., `pkg/util/cirrusutil/service.go`), NOT in API handler files.
- Use the Params/Result pattern for service functions (similar to gRPC):
  - Define a `*Params` struct containing all input parameters
  - Define a `*Result` struct containing all outputs (not including errors)
  - Example: `DeleteFiles(params DeleteFilesParams) (DeleteFilesResult, error)`
- This pattern ensures:
  - Business logic is testable in isolation without HTTP context
  - Service functions are reusable across different parts of the codebase (API, CLI, background jobs)
  - Clear separation of concerns between HTTP layer and domain logic
- See `pkg/util/cirrusutil/service.go` and `internal/server/api/v1/cirrus/` for reference implementations.

