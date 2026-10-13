# Layout integration

`App.vue` mounts `WorkspaceHost`. That host renders `AdminShell` or `UserShell` when the route meta `appLayout` is set, and renders the page directly otherwise.

User pages such as API keys, usage, account, redemption, and subscriptions live in `src/views/portal`. Administrator group management lives in `src/views/admin/hk/GroupsView.vue`. Routes are registered in `src/router/index.ts` and adjusted by `src/hk/workspaceRoutes.ts`.

Do not restore `AppHeader`, `AppLayout`, or `AppSidebar`. Those paths are listed in `hk/upstream-removed.json`.
