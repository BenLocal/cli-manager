<script setup lang="ts">
import Card from 'primevue/card'
import Tree from 'primevue/tree'
import type { TreeExpandedKeys } from 'primevue/tree'
import { computed } from 'vue'

import type { SessionItem } from '../types/console'

const props = defineProps<{
  session: SessionItem
}>()

const expandedKeys = computed<TreeExpandedKeys>(() =>
  props.session.files.reduce<TreeExpandedKeys>((acc, node) => {
    if (node.key) acc[String(node.key)] = true
    return acc
  }, {}),
)
</script>

<template>
  <Card class="workspace-card">
    <template #title>
      <div class="section-title section-title--workspace">
        <span>工作空间</span>
      </div>
    </template>

    <template #content>
      <Tree :value="session.files" :expanded-keys="expandedKeys" class="workspace-card__tree">
        <template #default="slotProps">
          <div class="workspace-node">
            <span>{{ slotProps.node.label }}</span>
            <small>{{ slotProps.node.data?.size ?? '' }}</small>
          </div>
        </template>
      </Tree>
    </template>
  </Card>
</template>
