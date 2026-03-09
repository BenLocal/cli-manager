<script setup lang="ts">
import Card from 'primevue/card'

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
            <div
              v-if="consoleState.showWorkspace.value"
              class="workbench"
              :class="{ 'workbench--expanded': !consoleState.showSidebar.value }"
            >
              <div class="workbench__main">
                <TerminalPanel
                  :session="consoleState.activeSession.value"
                  :node-name="consoleState.selectedNode.value.name"
                  :node-user="consoleState.selectedNode.value.user"
                  @submit-command="consoleState.submitCommand"
                />
              </div>
              <div class="workbench__side">
                <WorkspaceTree :session="consoleState.activeSession.value" />
              </div>
            </div>

            <TerminalPanel
              v-else
              :session="consoleState.activeSession.value"
              :node-name="consoleState.selectedNode.value.name"
              :node-user="consoleState.selectedNode.value.user"
              @submit-command="consoleState.submitCommand"
            />
          </template>

          <SessionGrid
            v-else
            :sessions="consoleState.selectedNodeState.value?.sessions ?? []"
            :active-session-id="consoleState.activeSession.value.id"
            @select-session="consoleState.switchSession"
            @close-session="consoleState.closeSession"
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
          </template>
        </Card>
      </section>
    </main>

    <SessionDialog
      v-model:visible="consoleState.showSessionDialog.value"
      :default-process="consoleState.selectedNode.value?.defaultProcess ?? 'bash'"
      :default-workspace="consoleState.selectedNode.value?.defaultWorkspace ?? '/root'"
      @create-session="consoleState.createSession"
    />
  </div>
</template>
