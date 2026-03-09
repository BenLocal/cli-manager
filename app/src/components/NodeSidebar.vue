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
  (event: 'update-node', value: { nodeId: string; payload: NewNodePayload }): void
  (event: 'delete-node', value: string): void
}>()

const showAddForm = ref(false)
const editingNodeId = ref('')
const form = reactive<NewNodePayload>({
  name: '',
  ip: '',
  port: '22',
  user: '',
  password: '',
})

function submitNode() {
  if (!form.name || !form.ip || !form.port || !form.user || !form.password) return
  if (editingNodeId.value) {
    emit('update-node', { nodeId: editingNodeId.value, payload: { ...form } })
  } else {
    emit('add-node', { ...form })
  }
  resetForm()
}

function startEdit(node: NodeItem) {
  editingNodeId.value = node.id
  showAddForm.value = true
  form.name = node.name
  form.ip = node.ip
  form.port = node.port
  form.user = node.user
  form.password = node.password
}

function resetForm() {
  editingNodeId.value = ''
  form.name = ''
  form.ip = ''
  form.port = '22'
  form.user = ''
  form.password = ''
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
            @click="showAddForm ? resetForm() : (showAddForm = true)"
          />
        </div>
      </template>

      <template #content>
        <div v-if="showAddForm" class="add-node-form">
          <InputText v-model="form.name" placeholder="节点名称" />
          <InputText v-model="form.ip" placeholder="IP 地址" />
          <InputText v-model="form.port" placeholder="端口" />
          <InputText v-model="form.user" placeholder="SSH 用户" />
          <InputText v-model="form.password" type="password" placeholder="密码" />
          <Button :label="editingNodeId ? '保存修改' : '确认添加'" @click="submitNode" />
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
                        <p>{{ node.user }}@{{ node.ip }}:{{ node.port }}</p>
                      </div>
                      <div class="node-card__actions">
                        <Button
                          class="node-card__action"
                          text
                          rounded
                          severity="secondary"
                          icon="pi pi-pencil"
                          @click.stop="startEdit(node)"
                        />
                        <Button
                          class="node-card__action node-card__action--danger"
                          text
                          rounded
                          severity="secondary"
                          icon="pi pi-trash"
                          @click.stop="emit('delete-node', node.id)"
                        />
                        <span class="node-card__dot" :class="statusClass(node.status)"></span>
                      </div>
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
