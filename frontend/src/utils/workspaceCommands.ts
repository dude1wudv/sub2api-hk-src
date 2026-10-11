import type { RouteLocationRaw } from 'vue-router'

export interface WorkspaceCommand {
  id: string
  label: string
  section: string
  keywords?: string
  to?: RouteLocationRaw
  href?: string
  run?: () => void | Promise<unknown>
}

/** Commands are supplied by the shell's permission-filtered navigation. */
export function searchWorkspaceCommands(commands: readonly WorkspaceCommand[], query: string) {
  const words = query.trim().toLocaleLowerCase().split(/\s+/).filter(Boolean)
  return commands.filter(command => words.every(word =>
    `${command.label} ${command.section} ${command.keywords || ''}`.toLocaleLowerCase().includes(word),
  ))
}

export function isWorkspaceShortcut(event: KeyboardEvent) {
  return (event.ctrlKey || event.metaKey) && !event.altKey && !event.shiftKey && event.key.toLowerCase() === 'k' && !event.isComposing
}
