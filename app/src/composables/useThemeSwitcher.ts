import { usePreset } from '@primeuix/themes'
import { onMounted, ref, watch } from 'vue'

import { themeOptions, themePresets } from '../themes/presets'
import type { ThemePresetKey } from '../types/console'

const THEME_STORAGE_KEY = 'cli-manager-theme'
const DARK_STORAGE_KEY = 'cli-manager-dark'

export function useThemeSwitcher() {
  const selectedTheme = ref<ThemePresetKey>('aura')
  const isDark = ref(true)
  const orderedThemes = themeOptions.map((item) => item.value)

  onMounted(() => {
    const savedTheme = window.localStorage.getItem(THEME_STORAGE_KEY) as ThemePresetKey | null
    const savedDark = window.localStorage.getItem(DARK_STORAGE_KEY)

    if (savedTheme && savedTheme in themePresets) {
      selectedTheme.value = savedTheme
    }

    if (savedDark !== null) {
      isDark.value = savedDark === 'true'
    }
  })

  watch(
    selectedTheme,
    (theme) => {
      usePreset(themePresets[theme])
      window.localStorage.setItem(THEME_STORAGE_KEY, theme)
    },
    { immediate: true },
  )

  watch(
    isDark,
    (value) => {
      document.documentElement.classList.toggle('app-dark', value)
      window.localStorage.setItem(DARK_STORAGE_KEY, String(value))
    },
    { immediate: true },
  )

  return {
    themeOptions,
    selectedTheme,
    isDark,
    cycleTheme: () => {
      const currentIndex = orderedThemes.indexOf(selectedTheme.value)
      const nextIndex = currentIndex === -1 ? 0 : (currentIndex + 1) % orderedThemes.length
      selectedTheme.value = orderedThemes[nextIndex] ?? orderedThemes[0] ?? 'aura'
    },
  }
}
