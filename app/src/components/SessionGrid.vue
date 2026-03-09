<script setup lang="ts">
import Button from 'primevue/button'
import Card from 'primevue/card'
import DataView from 'primevue/dataview'

import type { SessionItem } from '../types/console'

defineProps<{
  sessions: SessionItem[]
  activeSessionId: string
}>()

const emit = defineEmits<{
  (event: 'select-session', value: string): void
  (event: 'close-session', value: string): void
  (event: 'create-session'): void
}>()
</script>

<template>
  <div class="session-grid-shell">
    <DataView :value="sessions" layout="grid">
      <template #grid="{ items }">
        <div class="session-grid">
          <Card
            v-for="session in items"
            :key="session.id"
            class="session-grid__item"
            :class="{ 'session-grid__item--active': session.id === activeSessionId }"
            @click="emit('select-session', session.id)"
          >
            <template #content>
              <div class="session-grid__head">
                <span class="session-grid__symbol">&gt;_</span>
                <i v-if="session.id === activeSessionId" class="pi pi-check"></i>
              </div>

              <div class="session-grid__body">
                <h4>{{ session.name }}</h4>
                <p>{{ session.workspace }}</p>
                <p>{{ session.status === 'live' ? '在线会话' : '已关闭历史' }}</p>
              </div>

              <Button
                v-if="sessions.length > 1"
                class="session-grid__close"
                text
                rounded
                severity="secondary"
                icon="pi pi-times"
                @click.stop="emit('close-session', session.id)"
              />
            </template>
          </Card>

          <button class="session-grid__create" type="button" @click="emit('create-session')">
            <i class="pi pi-plus-circle"></i>
            <span>新建会话</span>
          </button>
        </div>
      </template>
    </DataView>
  </div>
</template>
