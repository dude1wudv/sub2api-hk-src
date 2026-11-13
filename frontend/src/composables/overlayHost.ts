import type { InjectionKey, Ref } from 'vue'

export const overlayHostKey: InjectionKey<Readonly<Ref<HTMLElement | null>>> = Symbol('overlay-host')
