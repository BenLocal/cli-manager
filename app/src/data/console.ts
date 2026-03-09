import type { TreeNode } from 'primevue/treenode'

import type { NodeItem, NodeSessionRecord } from '../types/console'

export const initialNodes: NodeItem[] = [
  {
    id: 'node-01',
    name: 'Primary-Alpha',
    status: 'online',
    ip: '192.168.1.10',
    port: '22',
    cpu: '12%',
    memory: '2.4GB',
    type: 'Core',
    user: 'admin',
    password: '',
    defaultProcess: 'bash',
    defaultWorkspace: '/srv/core',
  },
  {
    id: 'node-02',
    name: 'Storage-Beta',
    status: 'online',
    ip: '192.168.1.11',
    port: '22',
    cpu: '5%',
    memory: '16.8GB',
    type: 'Storage',
    user: 'root',
    password: '',
    defaultProcess: 'python3',
    defaultWorkspace: '/var/data',
  },
  {
    id: 'node-03',
    name: 'Edge-Worker-01',
    status: 'warning',
    ip: '192.168.5.2',
    port: '22',
    cpu: '88%',
    memory: '1.1GB',
    type: 'Worker',
    user: 'deploy',
    password: '',
    defaultProcess: 'node',
    defaultWorkspace: '/opt/app',
  },
]

export function createNodeSessions(): NodeSessionRecord {
  return Object.fromEntries(
    initialNodes.map((node, index) => {
      const sessionId = `sess-${node.id}-1`

      return [
        node.id,
        {
          activeSessionId: sessionId,
          sessions: [
            {
              id: sessionId,
              name: node.defaultProcess,
              workspace: node.defaultWorkspace,
              createdAt: `0${index + 8}:15:2${index}`,
              history: [
                `[SYSTEM] 成功连接至节点 ${node.name} (${node.ip})`,
                `[SYSTEM] 工作目录已切换至: ${node.defaultWorkspace}`,
                `[EXEC] 正在启动初始化进程: ${node.defaultProcess}...`,
              ],
              files: createWorkspaceTree(node.defaultWorkspace),
              status: 'live',
            },
          ],
        },
      ]
    }),
  )
}

export function createWorkspaceTree(rootPath: string): TreeNode[] {
  return [
    {
      key: `${rootPath}-src`,
      label: 'src',
      icon: 'pi pi-folder',
      children: [
        leaf(`${rootPath}-src-app`, 'app.go', '4.2KB'),
        leaf(`${rootPath}-src-handler`, 'handler.go', '2.8KB'),
        leaf(`${rootPath}-src-scheduler`, 'scheduler.ts', '6.1KB'),
      ],
    },
    {
      key: `${rootPath}-configs`,
      label: 'configs',
      icon: 'pi pi-folder',
      children: [
        leaf(`${rootPath}-configs-prod`, 'prod.yaml', '1.2KB'),
        leaf(`${rootPath}-configs-secrets`, 'secrets.env', '321B'),
      ],
    },
    leaf(`${rootPath}-readme`, 'README.md', '3.0KB'),
    leaf(`${rootPath}-log`, `${rootPath.split('/').join('_')}.log`, '8.4KB'),
  ]
}

function leaf(key: string, label: string, size: string): TreeNode {
  return {
    key,
    label,
    icon: 'pi pi-file',
    leaf: true,
    data: { size },
  }
}
