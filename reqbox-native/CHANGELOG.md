# Changelog

## 0.1.1 — 2026-11-09

Native UI port of Reqbox (mygo ui), aligned to the web baseline panel by
panel (batches 19–31 of the migration record):

- Request editor: KV grid (params/headers/form), Body/Auth panels, publish
  panel, tab panel scrolling, xl side-by-side layout.
- Response pane rebuilt on the web's cF: status bar, four line tabs
  (headers/request/preview/raw), collapsible General/headers/timing, editors
  that grow with the body.
- History tab with its own search and relative times; sidebar tree with
  foldable collections, kebab menus and published dots; empty state on the
  web's SF.
- Dialogs on the shadcn baseline: confirm, prompt, curl, run settings, and
  the 768px two-pane environment manager (save visibility fixed).
- Keyboard paths: input focus rings and a keyboard-reachable row delete.
- Dark window: DWM frame border follows the app theme (no white edge in
  dark mode).
- Packaging: mygo.json, signed updates (tagPrefix `reqbox-native-v`) and the
  NSIS installer.

## 0.1.0 — 2026-11-05

First native shell: theme base, sidebar, request bar, send pipeline,
response, history, mocks, environments, dialogs, i18n, settings, dark mode.
