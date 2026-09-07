export interface NavigationCommand {
  path: string
  label: string
  group: string
  run: () => void
}
