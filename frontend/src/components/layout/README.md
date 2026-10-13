# Layout components

The workspace shell is chosen in `src/hk/WorkspaceHost.vue`.

- `UserShell.vue` wraps signed-in user pages.
- `AdminShell.vue` wraps administrator pages.
- `AuthLayout.vue` wraps login and registration.
- `TablePageLayout.vue` is the shared table page frame.

`appLayout` on a route decides whether a shell is used. Administrator versus user is decided by `isAdminWorkspaceRoute`.
