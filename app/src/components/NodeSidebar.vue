<script setup lang="ts">
import Button from 'primevue/button'
import Card from 'primevue/card'
import DataView from 'primevue/dataview'
import Divider from 'primevue/divider'
import InputText from 'primevue/inputtext'
import { reactive, ref } from 'vue'

import type { ConsoleStats, NewNodePayload, NodeItem } from '../types/console'

const props = defineProps<{
  nodes: NodeItem[]
  selectedNodeId: string
  stats: ConsoleStats
}>()

const emit = defineEmits<{
  (event: 'select-node', value: string): void
  (event: 'add-node', value: NewNodePayload): void
}>()

const showAddForm = ref(false)
const form = reactive<NewNodePayload>({
  name: '',
  ip: '',
  user: '',
  defaultProcess: 'bash',
  defaultWorkspace: '/root',
})

function submitNode() {
  if (!form.name || !form.ip || !form.user) return
  emit('add-node', { ...form })
  form.name = ''
  form.ip = ''
  form.user = ''
  form.defaultProcess = 'bash'
  form.defaultWorkspace = '/root'
  showAddForm.value = false
}

function statusClass(status: NodeItem['status']) {
  if (status === 'online') return 'node-card__dot--online'
  if (status === 'warning') return 'node-card__dot--warning'
  return 'node-card__dot--offline'
}
</script>

<template>
  <aside class="node-sidebar">
    <Card class="node-sidebar__nodes">
      <template #title>
        <div class="sidebar-title">
          <div>
            <p class="app-eyebrow">节点列表</p>
          </div>
          <Button
            rounded
            :icon="showAddForm ? 'pi pi-minus' : 'pi pi-plus'"
            @click="showAddForm = !showAddForm"
          />
        </div>
      </template>

      <template #content>
        <div v-if="showAddForm" class="add-node-form">
          <InputText v-model="form.name" placeholder="节点名称" />
          <InputText v-model="form.ip" placeholder="IP 地址" />
          <InputText v-model="form.user" placeholder="SSH 用户" />
          <InputText v-model="form.defaultProcess" placeholder="默认进程" />
          <InputText v-model="form.defaultWorkspace" placeholder="默认工作区" />
          <Button label="确认添加" @click="submitNode" />
          <Divider />
        </div>

        <DataView :value="props.nodes" data-key="id" layout="list">
          <template #empty>
            <div class="empty-inline">没有匹配到节点。</div>
          </template>

          <template #list="{ items }">
            <div class="node-list">
              <Card
                v-for="node in items"
                :key="node.id"
                class="node-card"
                :class="{ 'node-card--active': node.id === selectedNodeId }"
              >
                <template #content>
                  <button class="node-card__button" type="button" @click="emit('select-node', node.id)">
                    <div class="node-card__head">
                      <div>
                        <h4>{{ node.name }}</h4>
                        <p>{{ node.user }}@{{ node.ip }}</p>
                      </div>
                      <span class="node-card__dot" :class="statusClass(node.status)"></span>
                    </div>
                  </button>
                </template>
              </Card>
            </div>
          </template>
        </DataView>
      </template>
    </Card>
  </aside>
</template>
