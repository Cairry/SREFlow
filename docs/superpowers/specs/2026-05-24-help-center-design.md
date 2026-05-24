# 帮助中心设计文档

## 概述

在导航栏新增 "Help" 菜单项，点击进入 `/help` 页面。页面采用单页面 + 左侧锚点导航布局，展示完整的帮助文档，包括快速入门、核心概念和节点参考。

## 架构

### 文件变更

| 文件 | 操作 | 说明 |
|------|------|------|
| `web/app/help/page.tsx` | 新增 | 帮助中心主页面 |
| `web/lib/help-content.ts` | 新增 | 帮助内容数据（中文） |
| `web/components/navigation.tsx` | 修改 | 添加 Help 导航项 |

### 路由

- 路径：`/help`
- 布局：左侧锚点导航（固定）+ 右侧内容区（可滚动）

## 页面结构

### 左侧导航栏

1. **快速入门** - 创建第一个工作流的步骤说明
2. **核心概念** - Workspaces、Workflows、Integrations、MCP Tools
3. **节点参考** - 按分类列出所有 14 个节点

### 右侧内容区

每个节点卡片包含：
- 节点名称 + 图标
- 功能简介（中文）
- 配置参数说明表格（字段名、类型、必填、说明）
- JSON 配置样例（代码块展示）

## 节点分类及内容

### Integration 类

1. **Prometheus** - 查询 Prometheus 指标
2. **Jaeger** - 查询分布式链路追踪
3. **Elasticsearch** - 搜索和分析日志
4. **Loki** - 使用 LogQL 查询日志
5. **ClickHouse** - 执行 SQL 查询
6. **Kubernetes** - 查询 Pod 状态和事件

### Action 类

7. **SSH** - 通过 SSH 执行远程命令
8. **GoScript** - 执行 GoScript 代码
9. **HTTP** - 发送 HTTP 请求

### AI 类

10. **LLM** - 调用大语言模型

### Logic 类

11. **Condition** - 条件判断和流程控制
12. **Start** - 工作流起点

### Output 类

13. **Output** - 输出工作流结果
14. **MCP** - 执行 MCP 工具（在 MCP Tools 页面配置后可用）

## 数据结构

`web/lib/help-content.ts` 中定义：

```typescript
interface HelpSection {
  id: string;           // 锚点 ID
  title: string;        // 标题
  content: ReactNode;   // 内容（JSX）
}

interface NodeHelpContent {
  type: string;         // 节点类型
  title: string;        // 中文名称
  category: string;     // 分类
  icon: string;         // 图标名
  description: string;  // 功能描述
  configFields: {
    name: string;
    type: string;
    required: boolean;
    description: string;
  }[];
  example: Record<string, unknown>;  // 配置样例
}
```

## UI 细节

- 左侧导航高亮当前滚动到的章节（基于 IntersectionObserver）
- 节点卡片使用 Card 组件，保持与现有 UI 风格一致
- 配置样例使用等宽字体代码块展示
- 响应式设计：移动端隐藏左侧导航，显示顶部目录
- 语言：全部中文
