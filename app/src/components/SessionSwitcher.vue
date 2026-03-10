<script setup lang="ts">
import Button from 'primevue/button'
import Card from 'primevue/card'
import Tab from 'primevue/tab'
import TabList from 'primevue/tablist'
import Tabs from 'primevue/tabs'

import type { SessionItem, ViewMode } from '../types/console'

const props = defineProps<{
  sessions: SessionItem[]
  activeSessionId: string
  viewMode: ViewMode
  sidebarVisible: boolean
}>()

const emit = defineEmits<{
  (event: 'switch-session', value: string): void
  (event: 'reconnect-session', value: string): void
  (event: 'update:view-mode', value: ViewMode): void
  (event: 'new-session'): void
}>()
</script>

<template>
  <Card class="session-switcher">
    <template #content>
      <div class="session-switcher__bar" :class="{ 'session-switcher__bar--full': !sidebarVisible }">
        <div class="session-switcher__tabs">
          <Button
            class="session-switcher__manager"
            :severity="viewMode === 'sessions' ? 'contrast' : 'secondary'"
            icon="pi pi-list"
            label="会话列表"
            @click="emit('update:view-mode', viewMode === 'sessions' ? 'terminal' : 'sessions')"
          />

          <Tabs :value="activeSessionId" scrollable @update:value="emit('switch-session', String($event))">
            <TabList>
              <Tab v-for="session in sessions" :key="session.id" :value="session.id">
                <span class="session-switcher__tab-label">
                  <span class="session-switcher__tab-text">
                    <i class="pi pi-angle-right"></i>
                    {{ session.name }}{{ session.status === 'closed' ? ' · closed' : '' }}
                  </span>
                  <Button
                    class="session-switcher__tab-action"
                    text
                    rounded
                    severity="secondary"
                    icon="pi pi-refresh"
                    :loading="session.status === 'connecting'"
                    :disabled="session.status === 'connecting'"
                    @click.stop="emit('reconnect-session', session.id)"
                  />
                </span>
              </Tab>
            </TabList>
          </Tabs>

          <Button
            class="session-switcher__create-right"
            text
            severity="secondary"
            rounded
            icon="pi pi-plus"
            @click="emit('new-session')"
          />
        </div>
      </div>
    </template>
  </Card>
</template>
