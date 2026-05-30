"use client";

import { useEffect, useState, useRef } from "react";
import {
  HelpCircle,
  Rocket,
  BookOpen,
  Activity,
  Search,
  BarChart3,
  Terminal,
  Code,
  Globe,
  Sparkles,
  GitBranch,
  FileOutput,
  Play,
  Puzzle,
  ChevronRight,
  Webhook,
  AlertTriangle,
} from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { NODE_HELP_CONTENT, NODE_CATEGORIES } from "@/lib/help-content";
import type { NodeHelpContent } from "@/lib/help-content";

const ICON_MAP: Record<string, React.ReactNode> = {
  activity: <Activity className="w-5 h-5" />,
  search: <Search className="w-5 h-5" />,
  "bar-chart-3": <BarChart3 className="w-5 h-5" />,
  terminal: <Terminal className="w-5 h-5" />,
  code: <Code className="w-5 h-5" />,
  globe: <Globe className="w-5 h-5" />,
  sparkles: <Sparkles className="w-5 h-5" />,
  "git-branch": <GitBranch className="w-5 h-5" />,
  "file-output": <FileOutput className="w-5 h-5" />,
  play: <Play className="w-5 h-5" />,
  puzzle: <Puzzle className="w-5 h-5" />,
};

const SECTIONS = [
  { id: "quickstart", label: "快速入门", icon: <Rocket className="w-4 h-4" /> },
  { id: "concepts", label: "核心概念", icon: <BookOpen className="w-4 h-4" /> },
  { id: "alertmanager", label: "对接 AlertManager", icon: <AlertTriangle className="w-4 h-4" /> },
  ...NODE_CATEGORIES.map((cat) => ({
    id: `node-${cat.id.toLowerCase()}`,
    label: cat.label,
    icon: <HelpCircle className="w-4 h-4" />,
    subItems: cat.nodes.map((n) => ({
      id: `node-${n.type.toLowerCase()}`,
      label: n.title,
    })),
  })),
];

function NodeHelpCard({ node }: { node: NodeHelpContent }) {
  return (
    <Card id={`node-${node.type.toLowerCase()}`} className="scroll-mt-24">
      <CardHeader>
        <div className="flex items-center gap-3">
          <div className="p-2 rounded-lg bg-primary/10 text-primary">
            {ICON_MAP[node.icon] || <HelpCircle className="w-5 h-5" />}
          </div>
          <div>
            <CardTitle className="text-lg">{node.title}</CardTitle>
            <Badge variant="outline" className="mt-1 text-xs">
              {node.category}
            </Badge>
          </div>
        </div>
      </CardHeader>
      <CardContent className="space-y-4">
        <p className="text-sm text-muted-foreground leading-relaxed">{node.description}</p>

        <div>
          <h4 className="text-sm font-semibold mb-2">配置参数</h4>
          <div className="rounded-lg border overflow-hidden">
            <table className="w-full text-sm">
              <thead>
                <tr className="bg-muted/50">
                  <th className="px-3 py-2 text-left font-medium">参数名</th>
                  <th className="px-3 py-2 text-left font-medium">类型</th>
                  <th className="px-3 py-2 text-left font-medium">必填</th>
                  <th className="px-3 py-2 text-left font-medium">说明</th>
                </tr>
              </thead>
              <tbody>
                {node.configFields.map((field) => (
                  <tr key={field.name} className="border-t">
                    <td className="px-3 py-2 font-mono text-xs">{field.name}</td>
                    <td className="px-3 py-2">
                      <Badge variant="secondary" className="text-xs">{field.type}</Badge>
                    </td>
                    <td className="px-3 py-2">
                      {field.required ? (
                        <Badge variant="destructive" className="text-xs">必填</Badge>
                      ) : (
                        <Badge variant="outline" className="text-xs">可选</Badge>
                      )}
                    </td>
                    <td className="px-3 py-2 text-muted-foreground">{field.description}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>

        <div>
          <h4 className="text-sm font-semibold mb-2">配置样例</h4>
          <div className="rounded-lg border bg-muted/20 p-4 space-y-3">
            {node.example.map((field) => (
              <div key={field.label} className="space-y-1">
                <label className="text-xs font-medium text-muted-foreground">{field.label}</label>
                {field.isCode ? (
                  <div className="rounded-md border bg-[#1e1e2e] text-[#cdd6f4] p-3 text-sm font-mono overflow-x-auto whitespace-pre">
                    {field.value}
                  </div>
                ) : (
                  <div className="rounded-md border bg-background px-3 py-2 text-sm font-mono">
                    {field.value}
                  </div>
                )}
              </div>
            ))}
          </div>
        </div>
      </CardContent>
    </Card>
  );
}

export default function HelpPage() {
  const [activeSection, setActiveSection] = useState("quickstart");
  const contentRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const headings = contentRef.current?.querySelectorAll("[id]");
    if (!headings?.length) return;

    const observer = new IntersectionObserver(
      (entries) => {
        for (const entry of entries) {
          if (entry.isIntersecting) {
            setActiveSection(entry.target.id);
          }
        }
      },
      { rootMargin: "-80px 0px -60% 0px" }
    );

    headings.forEach((h) => observer.observe(h));
    return () => observer.disconnect();
  }, []);

  const scrollToSection = (id: string) => {
    const el = document.getElementById(id);
    if (el) {
      el.scrollIntoView({ behavior: "smooth" });
      setActiveSection(id);
    }
  };

  return (
    <div className="min-h-screen bg-background">
      <div className="max-w-7xl mx-auto px-4 py-8">
        <div className="flex gap-8">
          {/* Left nav */}
          <nav className="w-56 shrink-0 hidden lg:block">
            <div className="sticky top-24 space-y-1">
              <h3 className="px-3 mb-3 text-sm font-semibold text-foreground">帮助中心</h3>
              {SECTIONS.map((section) => (
                <div key={section.id}>
                  <button
                    type="button"
                    onClick={() => scrollToSection(section.id)}
                    className={`w-full flex items-center gap-2 px-3 py-1.5 rounded-md text-sm transition-colors text-left ${
                      activeSection === section.id
                        ? "bg-primary/10 text-primary font-medium"
                        : "text-muted-foreground hover:text-foreground hover:bg-muted/50"
                    }`}
                  >
                    {section.icon}
                    <span className="truncate">{section.label}</span>
                  </button>
                  {section.subItems?.map((sub) => (
                    <button
                      key={sub.id}
                      type="button"
                      onClick={() => scrollToSection(sub.id)}
                      className={`w-full flex items-center gap-2 pl-8 pr-3 py-1 rounded-md text-xs transition-colors text-left ${
                        activeSection === sub.id
                          ? "bg-primary/10 text-primary font-medium"
                          : "text-muted-foreground hover:text-foreground hover:bg-muted/50"
                      }`}
                    >
                      <ChevronRight className="w-3 h-3" />
                      <span className="truncate">{sub.label}</span>
                    </button>
                  ))}
                </div>
              ))}
            </div>
          </nav>

          {/* Right content */}
          <div ref={contentRef} className="flex-1 min-w-0 space-y-8">
            {/* Quick Start */}
            <section id="quickstart" className="scroll-mt-24">
              <h2 className="text-2xl font-bold mb-4">快速入门</h2>
              <div className="space-y-4">
                <Card>
                  <CardContent className="pt-6">
                    <h3 className="font-semibold mb-3">1. 创建工作区</h3>
                    <p className="text-sm text-muted-foreground">
                      在导航栏的工作区选择器中，点击 &quot;Create First Workspace&quot; 创建你的第一个工作区。工作区用于组织和管理你的工作流和数据集成。
                    </p>
                  </CardContent>
                </Card>
                <Card>
                  <CardContent className="pt-6">
                    <h3 className="font-semibold mb-3">2. 配置集成</h3>
                    <p className="text-sm text-muted-foreground">
                      在 Integrations 页面配置你需要的数据源连接，如 Prometheus、Elasticsearch、SSH 等。配置完成后，这些连接即可在工作流中使用。
                    </p>
                  </CardContent>
                </Card>
                <Card>
                  <CardContent className="pt-6">
                    <h3 className="font-semibold mb-3">3. 创建工作流</h3>
                    <p className="text-sm text-muted-foreground">
                      在 Workflows 页面点击创建新工作流，通过拖拽节点到画布来编排自动化流程。从 Start 节点开始，按需添加数据查询、AI 分析、条件判断等节点，最后以 Output 节点输出结果。
                    </p>
                  </CardContent>
                </Card>
                <Card>
                  <CardContent className="pt-6">
                    <h3 className="font-semibold mb-3">4. 运行和调试</h3>
                    <p className="text-sm text-muted-foreground">
                      点击工具栏的运行按钮执行工作流。运行过程中可以实时查看每个节点的执行状态和输出结果。如果某个节点失败，可以查看错误信息并调整配置后重新运行。
                    </p>
                  </CardContent>
                </Card>
              </div>
            </section>

            {/* Core Concepts */}
            <section id="concepts" className="scroll-mt-24">
              <h2 className="text-2xl font-bold mb-4">核心概念</h2>
              <div className="space-y-4">
                <Card>
                  <CardHeader>
                    <CardTitle className="text-base">Workspace 工作区</CardTitle>
                  </CardHeader>
                  <CardContent>
                    <p className="text-sm text-muted-foreground">
                      工作区是组织和管理资源的基本单元。每个工作区拥有独立的工作流、集成配置和 MCP 工具。你可以在导航栏切换工作区，不同工作区之间的数据互不影响。
                    </p>
                  </CardContent>
                </Card>
                <Card>
                  <CardHeader>
                    <CardTitle className="text-base">Workflow 工作流</CardTitle>
                  </CardHeader>
                  <CardContent>
                    <p className="text-sm text-muted-foreground">
                      工作流是由多个节点按逻辑连接而成的自动化流程。每个工作流从 Start 节点开始，经过数据查询、AI 分析、条件判断等处理步骤，最终通过 Output 节点输出结果。节点之间通过连线传递数据，上游节点的输出可作为下游节点的输入。
                    </p>
                  </CardContent>
                </Card>
                <Card>
                  <CardHeader>
                    <CardTitle className="text-base">Integrations 集成</CardTitle>
                  </CardHeader>
                  <CardContent>
                    <p className="text-sm text-muted-foreground">
                      集成管理外部系统的连接配置，包括 Prometheus、Elasticsearch、SSH 等数据源和服务的连接信息。配置好的集成可在工作流的对应节点中直接引用，无需重复填写连接参数。
                    </p>
                  </CardContent>
                </Card>
                <Card>
                  <CardHeader>
                    <CardTitle className="text-base">MCP Tools MCP 工具</CardTitle>
                  </CardHeader>
                  <CardContent>
                    <p className="text-sm text-muted-foreground">
                      MCP（Model Context Protocol）工具是对外部工具服务的标准化接入。在 MCP Tools 页面配置 MCP 服务器后，启用其提供的工具即可在工作流中使用 MCP 节点调用这些工具，扩展工作流的能力边界。
                    </p>
                  </CardContent>
                </Card>
              </div>
            </section>

            {/* AlertManager Integration Guide */}
            <section id="alertmanager" className="scroll-mt-24">
              <h2 className="text-2xl font-bold mb-4 flex items-center gap-2">
                <AlertTriangle className="w-6 h-6 text-amber-500" />
                对接 AlertManager
              </h2>
              <div className="space-y-4">
                <Card>
                  <CardContent className="pt-6 space-y-3">
                    <p className="text-sm text-muted-foreground leading-relaxed">
                      AlertManager 是 Prometheus 生态中的告警管理组件。通过将 AlertManager 的 Webhook 接收器指向 RCAgent 工作流的执行接口，
                      可以在告警触发时自动执行根因分析工作流，实现 <strong className="text-foreground">告警 → 自动排查</strong> 的闭环。
                    </p>
                    <div className="rounded-lg border border-amber-500/30 bg-amber-500/5 p-3">
                      <p className="text-sm font-medium text-amber-600 dark:text-amber-400 mb-1">💡 工作原理</p>
                      <p className="text-xs text-muted-foreground">
                        AlertManager 发送告警 → Webhook 触发工作流执行（mode=json）→ Start 节点接收完整告警 JSON →
                        下游节点通过 <code className="text-violet-400 font-mono">start.status</code>、<code className="text-violet-400 font-mono">start.alerts</code> 等引用告警字段 → AI 自动分析根因
                      </p>
                    </div>
                  </CardContent>
                </Card>

                <Card>
                  <CardHeader>
                    <CardTitle className="text-base flex items-center gap-2">
                      <span className="inline-flex items-center justify-center w-6 h-6 rounded-full bg-primary/10 text-primary text-xs font-bold">1</span>
                      创建告警处理工作流
                    </CardTitle>
                  </CardHeader>
                  <CardContent className="space-y-2">
                    <p className="text-sm text-muted-foreground">
                      在 Workflows 页面新建一个工作流，按需编排节点。一个典型的告警分析工作流如下：
                    </p>
                    <div className="rounded-lg border bg-muted/20 p-4 space-y-2 text-sm">
                      <div className="flex items-center gap-2">
                        <span className="inline-flex items-center px-2 py-0.5 rounded bg-emerald-500/20 text-emerald-400 font-mono text-xs">Start</span>
                        <span className="text-muted-foreground">→ 接收 AlertManager 发送的告警 JSON</span>
                      </div>
                      <div className="flex items-center gap-2">
                        <span className="inline-flex items-center px-2 py-0.5 rounded bg-blue-500/20 text-blue-400 font-mono text-xs">SmartRouter</span>
                        <span className="text-muted-foreground">→ AI 自动分类告警类型（如：资源瓶颈 / 服务异常 / 配置变更）</span>
                      </div>
                      <div className="flex items-center gap-2">
                        <span className="inline-flex items-center px-2 py-0.5 rounded bg-violet-500/20 text-violet-400 font-mono text-xs">Prometheus / Loki / K8s</span>
                        <span className="text-muted-foreground">→ 查询对应系统的指标、日志和资源状态</span>
                      </div>
                      <div className="flex items-center gap-2">
                        <span className="inline-flex items-center px-2 py-0.5 rounded bg-pink-500/20 text-pink-400 font-mono text-xs">LLM</span>
                        <span className="text-muted-foreground">→ AI 分析告警原因并生成排查建议</span>
                      </div>
                      <div className="flex items-center gap-2">
                        <span className="inline-flex items-center px-2 py-0.5 rounded bg-orange-500/20 text-orange-400 font-mono text-xs">Output</span>
                        <span className="text-muted-foreground">→ 输出分析报告</span>
                      </div>
                    </div>
                    <p className="text-xs text-muted-foreground">
                      💡 关键：下游节点通过 <code className="text-violet-400 font-mono">{"{{start.alerts}}"}</code>、<code className="text-violet-400 font-mono">{"{{start.status}}"}</code> 等引用 AlertManager 发送的告警字段。
                    </p>
                  </CardContent>
                </Card>

                <Card>
                  <CardHeader>
                    <CardTitle className="text-base flex items-center gap-2">
                      <span className="inline-flex items-center justify-center w-6 h-6 rounded-full bg-primary/10 text-primary text-xs font-bold">2</span>
                      获取 Webhook URL
                    </CardTitle>
                  </CardHeader>
                  <CardContent className="space-y-2">
                    <p className="text-sm text-muted-foreground">
                      保存工作流后，点击编辑器工具栏中的 <strong className="text-foreground">API 调用</strong> 按钮，可以看到该工作流的执行接口。
                      AlertManager 需要使用的 Webhook URL 格式为：
                    </p>
                    <div className="rounded-md border bg-[#1e1e2e] text-[#cdd6f4] p-3 text-sm font-mono overflow-x-auto whitespace-pre">
                      {"{RCAgent地址}"}/api/workflow/{"{workflowId}"}/execute?mode=json
                    </div>
                    <div className="rounded-lg border border-blue-500/30 bg-blue-500/5 p-3 space-y-1">
                      <p className="text-sm font-medium text-blue-600 dark:text-blue-400">参数说明</p>
                      <ul className="text-xs text-muted-foreground space-y-1 list-disc list-inside">
                        <li><code className="text-violet-400 font-mono">mode=json</code> — 必须使用 json 模式，这样 AlertManager 发送的完整告警 JSON 会作为 Start 节点的输出</li>
                        <li><code className="text-violet-400 font-mono">workflowId</code> — 工作流 ID，保存工作流后自动生成，可在 API 调用对话框中查看</li>
                      </ul>
                    </div>
                  </CardContent>
                </Card>

                <Card>
                  <CardHeader>
                    <CardTitle className="text-base flex items-center gap-2">
                      <span className="inline-flex items-center justify-center w-6 h-6 rounded-full bg-primary/10 text-primary text-xs font-bold">3</span>
                      配置 AlertManager Webhook
                    </CardTitle>
                  </CardHeader>
                  <CardContent className="space-y-3">
                    <p className="text-sm text-muted-foreground">
                      在 AlertManager 的配置文件 <code className="font-mono text-xs bg-muted px-1 py-0.5 rounded">alertmanager.yml</code> 中添加 Webhook 接收器，将告警转发到 RCAgent：
                    </p>
                    <div className="rounded-md border bg-[#1e1e2e] text-[#cdd6f4] p-4 text-sm font-mono overflow-x-auto whitespace-pre">
{`route:
  receiver: "rcaagent-webhook"
  # 可按需配置路由规则，只将特定告警发送到 RCAgent
  # matchers:
  #   - severity = "critical"

receivers:
  - name: "rcaagent-webhook"
    webhook_configs:
      - url: "https://your-rcaagent-host/api/workflow/YOUR_WORKFLOW_ID/execute?mode=json"
        http_config:
          headers:
            X-Workspace-Id: "YOUR_WORKSPACE_ID"`}
                    </div>
                    <div className="rounded-lg border border-rose-500/30 bg-rose-500/5 p-3 space-y-1">
                      <p className="text-sm font-medium text-rose-600 dark:text-rose-400">⚠️ 必须配置的项</p>
                      <ul className="text-xs text-muted-foreground space-y-1 list-disc list-inside">
                        <li><strong>url</strong>：替换为你的 RCAgent 地址和工作流 ID</li>
                        <li><strong>X-Workspace-Id</strong>：在 <code className="font-mono">http_config.headers</code> 中添加，值为你的工作区 ID（在导航栏的工作区选择器中可以查看）</li>
                        <li>URL 中必须包含 <code className="text-violet-400 font-mono">?mode=json</code> 参数</li>
                      </ul>
                    </div>
                  </CardContent>
                </Card>

                <Card>
                  <CardHeader>
                    <CardTitle className="text-base flex items-center gap-2">
                      <span className="inline-flex items-center justify-center w-6 h-6 rounded-full bg-primary/10 text-primary text-xs font-bold">4</span>
                      在工作流中引用告警数据
                    </CardTitle>
                  </CardHeader>
                  <CardContent className="space-y-3">
                    <p className="text-sm text-muted-foreground">
                      使用 <code className="text-violet-400 font-mono">mode=json</code> 时，AlertManager 发送的完整告警 JSON 会作为 Start 节点的输出。
                      你可以在下游节点中通过变量引用这些数据。AlertManager Webhook 发送的 JSON 包含以下关键字段：
                    </p>
                    <div className="rounded-lg border overflow-hidden">
                      <table className="w-full text-sm">
                        <thead>
                          <tr className="bg-muted/50">
                            <th className="px-3 py-2 text-left font-medium">字段路径</th>
                            <th className="px-3 py-2 text-left font-medium">类型</th>
                            <th className="px-3 py-2 text-left font-medium">说明</th>
                          </tr>
                        </thead>
                        <tbody>
                          <tr className="border-t">
                            <td className="px-3 py-2 font-mono text-xs text-violet-400">status</td>
                            <td className="px-3 py-2"><Badge variant="secondary" className="text-xs">string</Badge></td>
                            <td className="px-3 py-2 text-muted-foreground">告警状态：firing 或 resolved</td>
                          </tr>
                          <tr className="border-t">
                            <td className="px-3 py-2 font-mono text-xs text-violet-400">alerts</td>
                            <td className="px-3 py-2"><Badge variant="secondary" className="text-xs">array</Badge></td>
                            <td className="px-3 py-2 text-muted-foreground">告警列表，包含每条告警的详细信息</td>
                          </tr>
                          <tr className="border-t">
                            <td className="px-3 py-2 font-mono text-xs text-violet-400">alerts[].labels</td>
                            <td className="px-3 py-2"><Badge variant="secondary" className="text-xs">object</Badge></td>
                            <td className="px-3 py-2 text-muted-foreground">告警标签，如 alertname、severity、namespace 等</td>
                          </tr>
                          <tr className="border-t">
                            <td className="px-3 py-2 font-mono text-xs text-violet-400">alerts[].annotations</td>
                            <td className="px-3 py-2"><Badge variant="secondary" className="text-xs">object</Badge></td>
                            <td className="px-3 py-2 text-muted-foreground">告警注解，如 summary、description 等</td>
                          </tr>
                          <tr className="border-t">
                            <td className="px-3 py-2 font-mono text-xs text-violet-400">alerts[].startsAt</td>
                            <td className="px-3 py-2"><Badge variant="secondary" className="text-xs">string</Badge></td>
                            <td className="px-3 py-2 text-muted-foreground">告警触发时间</td>
                          </tr>
                          <tr className="border-t">
                            <td className="px-3 py-2 font-mono text-xs text-violet-400">groupLabels</td>
                            <td className="px-3 py-2"><Badge variant="secondary" className="text-xs">object</Badge></td>
                            <td className="px-3 py-2 text-muted-foreground">分组标签</td>
                          </tr>
                          <tr className="border-t">
                            <td className="px-3 py-2 font-mono text-xs text-violet-400">externalURL</td>
                            <td className="px-3 py-2"><Badge variant="secondary" className="text-xs">string</Badge></td>
                            <td className="px-3 py-2 text-muted-foreground">AlertManager 外部访问地址</td>
                          </tr>
                        </tbody>
                      </table>
                    </div>
                    <p className="text-sm text-muted-foreground">
                      在节点配置中引用示例：
                    </p>
                    <div className="rounded-md border bg-[#1e1e2e] text-[#cdd6f4] p-3 text-sm font-mono overflow-x-auto whitespace-pre">
{`# 在 LLM 节点 prompt 中引用告警信息
当前告警状态: {{start.status}}
告警名称: {{start.alerts[0].labels.alertname}}
严重程度: {{start.alerts[0].labels.severity}}
命名空间: {{start.alerts[0].labels.namespace}}
告警描述: {{start.alerts[0].annotations.description}}
触发时间: {{start.alerts[0].startsAt}}`}
                    </div>
                  </CardContent>
                </Card>

                <Card>
                  <CardHeader>
                    <CardTitle className="text-base flex items-center gap-2">
                      <span className="inline-flex items-center justify-center w-6 h-6 rounded-full bg-primary/10 text-primary text-xs font-bold">5</span>
                      验证对接
                    </CardTitle>
                  </CardHeader>
                  <CardContent className="space-y-2">
                    <p className="text-sm text-muted-foreground">
                      配置完成后，可以通过以下方式验证：
                    </p>
                    <ol className="text-sm text-muted-foreground space-y-2 list-decimal list-inside">
                      <li>
                        <strong className="text-foreground">手动触发测试</strong>：在编辑器工具栏点击「API 调用」，复制 cURL 命令，
                        将 body 替换为 AlertManager 格式的 JSON（见下方示例），在终端执行确认工作流能正常运行
                      </li>
                      <li>
                        <strong className="text-foreground">触发真实告警</strong>：在 Prometheus 中配置一条测试告警规则（如模拟 high severity），
                        等待 AlertManager 触发，查看 RCAgent 工作流是否自动执行
                      </li>
                      <li>
                        <strong className="text-foreground">查看执行记录</strong>：在 Workflows 页面的运行历史中查看每次告警触发的工作流执行结果
                      </li>
                    </ol>
                    <div className="rounded-md border bg-muted/20 p-3 space-y-1">
                      <p className="text-xs font-medium text-muted-foreground mb-2">测试用的 AlertManager 请求体示例：</p>
                      <div className="rounded-md border bg-[#1e1e2e] text-[#cdd6f4] p-3 text-xs font-mono overflow-x-auto whitespace-pre">
{`{
  "status": "firing",
  "alerts": [{
    "status": "firing",
    "labels": {
      "alertname": "HighErrorRate",
      "severity": "critical",
      "namespace": "production",
      "service": "api-server"
    },
    "annotations": {
      "summary": "API 服务错误率过高",
      "description": "api-server 5xx 错误率超过 10%，持续 5 分钟"
    },
    "startsAt": "2026-05-30T08:00:00Z"
  }]
}`}
                      </div>
                    </div>
                  </CardContent>
                </Card>
              </div>
            </section>

            {/* Node reference sections */}
            {NODE_CATEGORIES.map((cat) => (
              <section key={cat.id} id={`node-${cat.id.toLowerCase()}`} className="scroll-mt-24">
                <h2 className="text-2xl font-bold mb-4">{cat.label}</h2>
                <div className="space-y-6">
                  {cat.nodes.map((node) => (
                    <NodeHelpCard key={node.type} node={node} />
                  ))}
                </div>
              </section>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}
