<script setup lang="ts">
import Button from 'primevue/button'
import Card from 'primevue/card'
import Splitter from 'primevue/splitter'
import SplitterPanel from 'primevue/splitterpanel'

import AppHeader from './components/AppHeader.vue'
import NodeSidebar from './components/NodeSidebar.vue'
import SessionDialog from './components/SessionDialog.vue'
import SessionGrid from './components/SessionGrid.vue'
import SessionSwitcher from './components/SessionSwitcher.vue'
import TerminalPanel from './components/TerminalPanel.vue'
import WorkspaceTree from './components/WorkspaceTree.vue'
import { useConsoleManager } from './composables/useConsoleManager'
import { useThemeSwitcher } from './composables/useThemeSwitcher'

const consoleState = useConsoleManager()
const themeState = useThemeSwitcher()
</script>

<template>
  <div class="app-shell">
    <AppHeader
      :node-count="consoleState.nodes.value.length"
      :online-count="consoleState.overviewStats.value.online"
      :session-count="consoleState.overviewStats.value.sessions"
      :selected-theme="themeState.selectedTheme.value"
      :dark-mode="themeState.isDark.value"
      :sidebar-visible="consoleState.showSidebar.value"
      @toggle-sidebar="consoleState.showSidebar.value = !consoleState.showSidebar.value"
      @cycle-theme="themeState.cycleTheme"
      @new-session="consoleState.openSessionDialog"
    />

    <main class="app-layout" :class="{ 'app-layout--expanded': !consoleState.showSidebar.value }">
      <NodeSidebar
        v-if="consoleState.showSidebar.value"
        :nodes="consoleState.nodes.value"
        :selected-node-id="consoleState.selectedNodeId.value"
        :stats="consoleState.overviewStats.value"
        @select-node="consoleState.selectNode"
        @add-node="consoleState.addNode"
        @update-node="consoleState.updateNode($event.nodeId, $event.payload)"
        @delete-node="consoleState.deleteNode"
      />

      <section class="app-layout__content">
        <template v-if="consoleState.selectedNode.value && consoleState.activeSession.value">
          <SessionSwitcher
            :sessions="consoleState.selectedNodeState.value?.sessions ?? []"
            :active-session-id="consoleState.activeSession.value.id"
            :view-mode="consoleState.viewMode.value"
            :sidebar-visible="consoleState.showSidebar.value"
            @switch-session="consoleState.switchSession"
            @update:view-mode="consoleState.viewMode.value = $event"
            @new-session="consoleState.openSessionDialog"
          />

          <template v-if="consoleState.viewMode.value === 'terminal'">
            <Splitter
              v-if="consoleState.showWorkspace.value"
              class="workbench"
              :class="{ 'workbench--expanded': !consoleState.showSidebar.value }"
              :gutter-size="8"
              layout="horizontal"
            >
              <SplitterPanel class="workbench__main" :size="76" :min-size="45">
                <TerminalPanel
                  :session="consoleState.activeSession.value"
                  :node-name="consoleState.selectedNode.value.name"
                  :node-user="consoleState.selectedNode.value.user"
                />
              </SplitterPanel>
              <SplitterPanel class="workbench__side" :size="24" :min-size="18">
                <WorkspaceTree :session="consoleState.activeSession.value" />
              </SplitterPanel>
            </Splitter>

            <TerminalPanel
              v-else
              :session="consoleState.activeSession.value"
              :node-name="consoleState.selectedNode.value.name"
              :node-user="consoleState.selectedNode.value.user"
            />
          </template>

          <SessionGrid
            v-else
            :sessions="consoleState.selectedNodeState.value?.sessions ?? []"
            :active-session-id="consoleState.activeSession.value.id"
            @select-session="consoleState.switchSession"
            @edit-session="consoleState.openSessionDialog"
            @delete-session="consoleState.deleteSession"
            @create-session="consoleState.openSessionDialog"
          />
        </template>

        <Card v-else class="empty-card">
          <template #content>
            <p class="app-eyebrow">{{ consoleState.selectedNode.value ? '等待会话' : '空闲工作台' }}</p>
            <h2>{{ consoleState.selectedNode.value ? '请先创建一个终端会话' : '请先选择一个节点' }}</h2>
            <p>
              {{
                consoleState.selectedNode.value
                  ? '顶部加号会通过 SignalR 创建后端 PTY 会话。'
                  : '左侧节点列表和顶部会话入口会在选中节点后激活。'
              }}
            </p>
            <Button
              v-if="consoleState.selectedNode.value"
              class="empty-card__action"
              rounded
              icon="pi pi-plus"
              label="添加会话"
              @click="() => consoleState.openSessionDialog()"
            />
          </template>
        </Card>
      </section>
    </main>

    <SessionDialog
      v-model:visible="consoleState.showSessionDialog.value"
      :mode="consoleState.sessionDialogMode.value"
      :default-process="consoleState.sessionDialogDefaults.value.process"
      :default-workspace="consoleState.sessionDialogDefaults.value.workspace"
      @save-session="consoleState.saveSession"
    />
  </div>
</template>
