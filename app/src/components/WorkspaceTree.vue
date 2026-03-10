<script setup lang="ts">
import Card from 'primevue/card'
import Tree from 'primevue/tree'
import type { TreeNode } from 'primevue/treenode'
import type { TreeExpandedKeys } from 'primevue/tree'
import { ref, watch } from 'vue'

import { listNodeFiles, type FileNodeDto } from '../api'
import type { SessionItem } from '../types/console'

const props = defineProps<{
  nodeId: string
  session: SessionItem
}>()

const nodes = ref<TreeNode[]>([])
const expandedKeys = ref<TreeExpandedKeys>({})

watch(
  () => [props.nodeId, props.session.workspace] as const,
  async ([nodeId, workspace]) => {
    nodes.value = await loadDirectory(nodeId, workspace)
    expandedKeys.value = {}
  },
  { immediate: true },
)

async function handleNodeExpand(node: TreeNode) {
  if (node.leaf || !node.data?.path || Array.isArray(node.children) && node.children.length > 0) {
    return
  }
  node.children = await loadDirectory(props.nodeId, String(node.data.path))
  nodes.value = [...nodes.value]
}

async function loadDirectory(nodeId: string, dirPath: string): Promise<TreeNode[]> {
  const items = await listNodeFiles(nodeId, dirPath)
  return items.map(mapFileNode)
}

function mapFileNode(item: FileNodeDto): TreeNode {
  return {
    key: item.key,
    label: item.label,
    icon: item.icon,
    leaf: item.leaf,
    children: item.leaf ? undefined : [],
    data: {
      path: item.path,
      size: item.size,
    },
  }
}
</script>

<template>
  <Card class="workspace-card">
    <template #title>
      <div class="section-title section-title--workspace">
        <span>工作空间</span>
      </div>
    </template>

    <template #content>
      <Tree
        v-model:expandedKeys="expandedKeys"
        :value="nodes"
        class="workspace-card__tree"
        @node-expand="handleNodeExpand"
      >
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
