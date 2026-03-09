import type { TreeNode } from 'primevue/treenode'

import type { NodeItem, NodeSessionRecord } from '../types/console'

export const initialNodes: NodeItem[] = [
  
]

export function createNodeSessions(): NodeSessionRecord {
  return {}
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
