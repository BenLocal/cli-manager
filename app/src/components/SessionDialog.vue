<script setup lang="ts">
import Button from 'primevue/button'
import Dialog from 'primevue/dialog'
import FloatLabel from 'primevue/floatlabel'
import InputText from 'primevue/inputtext'
import { reactive, watch } from 'vue'

const props = defineProps<{
  visible: boolean
  defaultProcess: string
  defaultWorkspace: string
}>()

const emit = defineEmits<{
  (event: 'update:visible', value: boolean): void
  (event: 'create-session', value: { process: string; workspace: string }): void
}>()

const form = reactive({
  process: '',
  workspace: '',
})

watch(
  () => [props.visible, props.defaultProcess, props.defaultWorkspace] as const,
  ([visible, process, workspace]) => {
    if (!visible) return
    form.process = process
    form.workspace = workspace
  },
  { immediate: true },
)

function close() {
  emit('update:visible', false)
}

function submit() {
  if (!form.process || !form.workspace) return
  emit('create-session', { process: form.process, workspace: form.workspace })
}
</script>

<template>
  <Dialog
    :visible="visible"
    modal
    header="初始化新会话"
    :style="{ width: 'min(92vw, 34rem)' }"
    @update:visible="emit('update:visible', $event)"
  >
    <div class="session-dialog">
      <FloatLabel variant="in">
        <InputText id="session-process" v-model="form.process" fluid />
        <label for="session-process">启动进程</label>
      </FloatLabel>

      <FloatLabel variant="in">
        <InputText id="session-workspace" v-model="form.workspace" fluid />
        <label for="session-workspace">工作区路径</label>
      </FloatLabel>

      <div class="session-dialog__actions">
        <Button label="取消" text severity="secondary" @click="close" />
        <Button label="启动会话" icon="pi pi-play" @click="submit" />
      </div>
    </div>
  </Dialog>
</template>
