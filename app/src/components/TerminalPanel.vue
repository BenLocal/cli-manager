<script setup lang="ts">
import Button from 'primevue/button'
import Card from 'primevue/card'
import InputGroup from 'primevue/inputgroup'
import InputGroupAddon from 'primevue/inputgroupaddon'
import InputText from 'primevue/inputtext'
import ScrollPanel from 'primevue/scrollpanel'
import { computed, ref } from 'vue'

import type { SessionItem } from '../types/console'

const props = defineProps<{
  session: SessionItem
  nodeName: string
  nodeUser: string
}>()

const emit = defineEmits<{
  (event: 'submit-command', value: string): void
}>()

const command = ref('')
const live = computed(() => props.session.status === 'live')

function submit() {
  const value = command.value.trim()
  if (!value || !live.value) return
  emit('submit-command', value)
  command.value = ''
}

function lineClass(line: string) {
  if (line.startsWith('[SYSTEM]')) return 'terminal-line terminal-line--system'
  if (line.startsWith('[EXEC]')) return 'terminal-line terminal-line--exec'
  return 'terminal-line'
}
</script>

<template>
  <Card class="terminal-card">
    <template #title>
      <div class="section-title section-title--terminal">
        <span>TTY: {{ session.name.toUpperCase() }}</span>
        <span>DIR: {{ session.workspace.toUpperCase() }}</span>
      </div>
    </template>

    <template #content>
      <ScrollPanel class="terminal-card__scroll">
        <div class="terminal-log">
          <div v-for="(line, index) in session.history" :key="`${session.id}-${index}`" :class="lineClass(line)">
            {{ line }}
          </div>
        </div>
      </ScrollPanel>

      <form class="terminal-card__form" @submit.prevent="submit">
        <InputGroup>
          <InputGroupAddon>➜</InputGroupAddon>
          <InputText
            v-model="command"
            :disabled="!live"
            :placeholder="live ? '输入控制指令...' : '当前会话已关闭，无法继续输入'"
          />
          <Button icon="pi pi-send" type="submit" :disabled="!live" />
        </InputGroup>
      </form>
    </template>
  </Card>
</template>
