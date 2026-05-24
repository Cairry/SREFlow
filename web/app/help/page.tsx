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
