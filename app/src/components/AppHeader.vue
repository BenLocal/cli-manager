<script setup lang="ts">
import Button from 'primevue/button'
import Toolbar from 'primevue/toolbar'
import type { ThemePresetKey } from '../types/console'

const props = defineProps<{
  nodeCount: number
  onlineCount: number
  sessionCount: number
  selectedTheme: ThemePresetKey
  darkMode: boolean
  sidebarVisible: boolean
}>()

const emit = defineEmits<{
  (event: 'toggle-sidebar'): void
  (event: 'cycle-theme'): void
  (event: 'new-session'): void
}>()
</script>

<template>
  <Toolbar class="app-toolbar">
    <template #start>
      <div class="toolbar-start">
        <Button
          class="toolbar-toggle"
          rounded
          text
          severity="secondary"
          :icon="sidebarVisible ? 'pi pi-angle-double-left' : 'pi pi-angle-double-right'"
          @click="emit('toggle-sidebar')"
        />

        <div class="toolbar-brand">
          <div class="toolbar-title">
            <span>CLI</span>
            <span class="toolbar-title__accent">MANAGER</span>
          </div>
        </div>
      </div>
    </template>

    <template #end>
      <div class="toolbar-end">
        <Button
          class="toolbar-theme"
          text
          severity="secondary"
          rounded
          :icon="darkMode ? 'pi pi-moon' : 'pi pi-sun'"
          @click="emit('cycle-theme')"
        />
      </div>
    </template>
  </Toolbar>
</template>
