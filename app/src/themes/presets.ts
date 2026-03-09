import { definePreset } from '@primeuix/themes'
import Aura from '@primevue/themes/aura'
import Lara from '@primevue/themes/lara'
import Material from '@primevue/themes/material'
import Nora from '@primevue/themes/nora'

import type { ThemeOption, ThemePresetKey } from '../types/console'

const cyanPrimary = {
  50: '#ecfeff',
  100: '#cffafe',
  200: '#a5f3fc',
  300: '#67e8f9',
  400: '#22d3ee',
  500: '#06b6d4',
  600: '#0891b2',
  700: '#0e7490',
  800: '#155e75',
  900: '#164e63',
  950: '#083344',
}

const emeraldPrimary = {
  50: '#ecfdf5',
  100: '#d1fae5',
  200: '#a7f3d0',
  300: '#6ee7b7',
  400: '#34d399',
  500: '#10b981',
  600: '#059669',
  700: '#047857',
  800: '#065f46',
  900: '#064e3b',
  950: '#022c22',
}

const amberPrimary = {
  50: '#fffbeb',
  100: '#fef3c7',
  200: '#fde68a',
  300: '#fcd34d',
  400: '#fbbf24',
  500: '#f59e0b',
  600: '#d97706',
  700: '#b45309',
  800: '#92400e',
  900: '#78350f',
  950: '#451a03',
}

const rosePrimary = {
  50: '#fff1f2',
  100: '#ffe4e6',
  200: '#fecdd3',
  300: '#fda4af',
  400: '#fb7185',
  500: '#f43f5e',
  600: '#e11d48',
  700: '#be123c',
  800: '#9f1239',
  900: '#881337',
  950: '#4c0519',
}

export const themePresets = {
  aura: definePreset(Aura, {
    semantic: { primary: cyanPrimary },
  }),
  lara: definePreset(Lara, {
    semantic: { primary: emeraldPrimary },
  }),
  nora: definePreset(Nora, {
    semantic: { primary: amberPrimary },
  }),
  material: definePreset(Material, {
    semantic: { primary: rosePrimary },
  }),
} satisfies Record<ThemePresetKey, unknown>

export const themeOptions: ThemeOption[] = [
  { label: 'Aura', value: 'aura' },
  { label: 'Lara', value: 'lara' },
  { label: 'Nora', value: 'nora' },
  { label: 'Material', value: 'material' },
]

export function getThemePreset(key: ThemePresetKey) {
  return themePresets[key]
}
