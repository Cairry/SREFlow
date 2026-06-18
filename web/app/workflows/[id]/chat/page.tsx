"use client";

import { useState, useEffect, useRef } from "react";
import { useParams, useRouter } from "next/navigation";
import { Textarea } from "@/components/ui/textarea";
import { ScrollArea } from "@/components/ui/scroll-area";
import { ArrowLeft, Send, Loader2, Brain, MessageSquare, ChevronDown, ChevronRight, Workflow, Sparkles, Trash2, PanelTop, PanelTopClose, Square, Plus, MessageCircle } from "lucide-react";
import { workflowApi, ChatSession, ChatMessage } from "@/lib/api/workflow";
import { cn } from "@/lib/utils";
import { isNavHidden, toggleNav } from "@/lib/nav-toggle";

interface ThinkingStep {
  nodeId: string;
  nodeType: string;
  content: string;
  elapsed_time?: string;
}

interface Message {
  id: string;
  role: "user" | "assistant";
  content: string;
  timestamp: Date;
  thinking?: ThinkingStep[];
}

interface StreamEvent {
  event: string;
  data: any;
  node?: {
    id: string;
    type: string;
    outputs: Record<string, string>;
    elapsed_time?: string;
  };
}

export default function ChatPage() {
  const params = useParams();
  const router = useRouter();
  const workflowId = params.id as string;
  
  const [messages, setMessages] = useState<Message[]>([]);
  const [input, setInput] = useState("");
  const [isLoading, setIsLoading] = useState(false);
  const [isStopping, setIsStopping] = useState(false);
  const [workflowName, setWorkflowName] = useState("");
  const [collapsedThinking, setCollapsedThinking] = useState<Set<string>>(new Set());
  const [navHidden, setNavHidden] = useState(isNavHidden());
  const [currentExecutionId, setCurrentExecutionId] = useState<string | null>(null);
  
  // Session management
  const [sessions, setSessions] = useState<ChatSession[]>([]);
  const [currentSession, setCurrentSession] = useState<ChatSession | null>(null);
  const [showSessionList, setShowSessionList] = useState(true);
  
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const abortControllerRef = useRef<AbortController | null>(null);

  useEffect(() => {
    loadWorkflow();
    loadSessions();
  }, [workflowId]);

  useEffect(() => {
    scrollToBottom();
  }, [messages]);
  
  useEffect(() => {
    const handler = () => setNavHidden(isNavHidden());
    window.addEventListener("storage", handler);
    window.addEventListener("flowforge-nav-toggle", handler);
    return () => {
      window.removeEventListener("storage", handler);
      window.removeEventListener("flowforge-nav-toggle", handler);
    };
  }, []);

  const loadWorkflow = async () => {
    try {
      const workflow = await workflowApi.getById(workflowId);
      setWorkflowName(workflow.name);
    } catch (error) {
      console.error("Failed to load workflow:", error);
    }
  };

  const loadSessions = async () => {
    try {
      const sessionList = await workflowApi.listChatSessions(workflowId);
      setSessions(sessionList || []);
      
      // If there are sessions, load the most recent one
      if (sessionList && sessionList.length > 0) {
        await selectSession(sessionList[0]);
      }
    } catch (error) {
      console.error("Failed to load sessions:", error);
    }
  };

  const selectSession = async (session: ChatSession) => {
    try {
      setCurrentSession(session);
      const messages = await workflowApi.getChatMessages(workflowId, session.id);
      
      const formattedMessages: Message[] = (messages || []).map(msg => {
        let thinking: ThinkingStep[] | undefined;
        if (msg.thinking) {
          try {
            const parsed = JSON.parse(msg.thinking);
            // Convert saved thinking format to display format
            thinking = parsed.map((step: any) => ({
              nodeId: step.nodeId,
              nodeType: step.nodeType,
              content: extractContent(step.outputs || {}),
              elapsed_time: step.elapsedTime,
            }));
          } catch (e) {
            console.error("Failed to parse thinking:", e);
          }
        }
        
        return {
          id: msg.id,
          role: msg.role,
          content: msg.content,
          thinking,
          timestamp: new Date(msg.createdAt),
        };
      });
      
      setMessages(formattedMessages);
    } catch (error) {
      console.error("Failed to load session messages:", error);
    }
  };

  const createNewSession = async () => {
    try {
      const session = await workflowApi.createChatSession(workflowId, `Chat ${new Date().toLocaleString()}`);
      setSessions(prev => [session, ...prev]);
      setCurrentSession(session);
      setMessages([]);
    } catch (error) {
      console.error("Failed to create session:", error);
    }
  };

  const deleteSession = async (sessionId: string, e: React.MouseEvent) => {
    e.stopPropagation();
    try {
      await workflowApi.deleteChatSession(workflowId, sessionId);
      setSessions(prev => prev.filter(s => s.id !== sessionId));
      
      if (currentSession?.id === sessionId) {
        const remaining = sessions.filter(s => s.id !== sessionId);
        if (remaining.length > 0) {
          await selectSession(remaining[0]);
        } else {
          setCurrentSession(null);
          setMessages([]);
        }
      }
    } catch (error) {
      console.error("Failed to delete session:", error);
    }
  };

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
  };

  const parseSSE = (text: string): StreamEvent[] => {
    const events: StreamEvent[] = [];
    const lines = text.split("\n");
    
    let currentEvent: Partial<StreamEvent> = {};
    
    for (const line of lines) {
      if (line.startsWith("event:")) {
        currentEvent.event = line.substring(7).trim();
      } else if (line.startsWith("data:")) {
        const dataStr = line.substring(5).trim();
        try {
          const parsed = JSON.parse(dataStr);
          currentEvent.data = parsed;
          // Extract node from parsed data (node is at top level in SSE)
          if (parsed.node) {
            currentEvent.node = parsed.node;
          }
        } catch {
          currentEvent.data = dataStr;
        }
      } else if (line === "" && currentEvent.event) {
        events.push(currentEvent as StreamEvent);
        currentEvent = {};
      }
    }
    
    return events;
  };

  const extractContent = (outputs: Record<string, string>): string => {
    if (outputs.reply) return outputs.reply;
    if (outputs.text) return outputs.text;
    if (outputs.result) return outputs.result;
    if (outputs.response) return outputs.response;
    return Object.entries(outputs)
      .map(([k, v]) => `[${k}]: ${v}`)
      .join("\n");
  };

  const toggleThinking = (messageId: string) => {
    setCollapsedThinking(prev => {
      const next = new Set(prev);
      if (next.has(messageId)) {
        next.delete(messageId);
      } else {
        next.add(messageId);
      }
      return next;
    });
  };

  const handleStop = async () => {
    if (!currentExecutionId || isStopping) return;
    setIsStopping(true);
    try {
      abortControllerRef.current?.abort();
      await workflowApi.cancelExecution(currentExecutionId);
    } catch (err) {
      console.error("[Stop] Error:", err);
    } finally {
      setIsStopping(false);
      setIsLoading(false);
      setCurrentExecutionId(null);
    }
  };

  const handleSend = async () => {
    if (!input.trim() || isLoading) return;

    // Create session if not exists
    let sessionId = currentSession?.id;
    if (!sessionId) {
      try {
        const session = await workflowApi.createChatSession(workflowId, input.trim().slice(0, 30));
        sessionId = session.id;
        setCurrentSession(session);
        setSessions(prev => [session, ...prev]);
      } catch (error) {
        console.error("Failed to create session:", error);
        return;
      }
    }

    const userMessage: Message = {
      id: `user-${Date.now()}`,
      role: "user",
      content: input.trim(),
      timestamp: new Date(),
    };

    setMessages((prev) => [...prev, userMessage]);
    setInput("");
    setIsLoading(true);

    const assistantMessageId = `assistant-${Date.now()}`;
    const assistantMessage: Message = {
      id: assistantMessageId,
      role: "assistant",
      content: "",
      thinking: [],
      timestamp: new Date(),
    };

    setMessages((prev) => [...prev, assistantMessage]);

    const abortController = new AbortController();
    abortControllerRef.current = abortController;

    try {
      const stream = await workflowApi.chat(workflowId, userMessage.content, 'text', abortController.signal, sessionId);
      const reader = stream.getReader();
      const decoder = new TextDecoder();

      let buffer = "";

      while (true) {
        const { done, value } = await reader.read();
        if (done) break;

        buffer += decoder.decode(value, { stream: true });
        const events = parseSSE(buffer);
        
        const lastNewline = buffer.lastIndexOf("\n\n");
        if (lastNewline !== -1) {
          buffer = buffer.substring(lastNewline + 2);
        }

        for (const event of events) {
          if (event.event === "heartbeat") continue;

          if (event.event === "execution_started" && event.data?.executionId) {
            setCurrentExecutionId(event.data.executionId);
            continue;
          }
          
          // Node info is at top level, not in data.node
          const node = event.node;
          const nodeId = node?.id;
          const nodeType = node?.type || "";
          const outputs = node?.outputs || {};
          const elapsedTime = node?.elapsed_time || "";

          if ((event.event === "thinking" || event.event === "reply") && nodeId) {
            const content = extractContent(outputs);
            
            if (event.event === "thinking") {
              setMessages((prev) =>
                prev.map((msg) => {
                  if (msg.id !== assistantMessageId) return msg;
                  
                  const existingStep = msg.thinking?.find(s => s.nodeId === nodeId);
                  if (existingStep) {
                    return {
                      ...msg,
                      thinking: msg.thinking!.map(s => 
                        s.nodeId === nodeId ? { ...s, content: s.content + content } : s
                      ),
                    };
                  }
                  
                  return {
                    ...msg,
                    thinking: [...(msg.thinking || []), { nodeId, nodeType, content, elapsed_time: elapsedTime }],
                  };
                })
              );
            } else {
              setMessages((prev) =>
                prev.map((msg) =>
                  msg.id === assistantMessageId
                    ? { ...msg, content: msg.content + content }
                    : msg
                )
              );
            }
          } else if (event.event === "workflow_completed") {
            setMessages((prev) =>
              prev.map((msg) =>
                msg.id === assistantMessageId
                  ? { ...msg, content: msg.content || "Workflow completed." }
                  : msg
              )
            );
          }
        }
      }
      
      setCollapsedThinking(prev => new Set(prev).add(assistantMessageId));
      
      // Reload sessions to update the order
      const updatedSessions = await workflowApi.listChatSessions(workflowId);
      setSessions(updatedSessions || []);
    } catch (error) {
      if (error instanceof DOMException && error.name === "AbortError") {
        setMessages((prev) =>
          prev.map((msg) =>
            msg.id === assistantMessageId
              ? { ...msg, content: msg.content || "[Stopped]" }
              : msg
          )
        );
      } else {
        console.error("Chat error:", error);
        setMessages((prev) =>
          prev.map((msg) =>
            msg.id === assistantMessageId
              ? { ...msg, content: `Error: ${error instanceof Error ? error.message : "Unknown error"}` }
              : msg
          )
        );
      }
    } finally {
      setIsLoading(false);
      setCurrentExecutionId(null);
      abortControllerRef.current = null;
    }
  };

  return (
    <div className="flex h-screen bg-background">
      {/* Session Sidebar */}
      {showSessionList && (
        <div className="w-64 border-r border-workflow-border bg-workflow-surface flex flex-col">
          <div className="p-3 border-b border-workflow-border">
            <button
              onClick={createNewSession}
              className="w-full flex items-center justify-center gap-2 px-3 py-2 bg-primary hover:bg-primary/90 text-primary-foreground rounded-lg transition-colors text-sm"
            >
              <Plus className="w-4 h-4" />
              New Chat
            </button>
          </div>
          
          <ScrollArea className="flex-1">
            <div className="p-2 space-y-1">
              {sessions.length === 0 && (
                <div className="text-center text-workflow-text-muted text-xs py-8">
                  No conversations yet
                </div>
              )}
              
              {sessions.map((session) => (
                <div
                  key={session.id}
                  onClick={() => selectSession(session)}
                  className={cn(
                    "group flex items-center gap-2 px-3 py-2 rounded-lg cursor-pointer transition-colors",
                    currentSession?.id === session.id
                      ? "bg-primary/10 text-primary"
                      : "hover:bg-workflow-surface-hover text-workflow-text"
                  )}
                >
                  <MessageCircle className="w-4 h-4 shrink-0" />
                  <div className="flex-1 min-w-0">
                    <div className="text-sm truncate">{session.title.length > 15 ? session.title.slice(0, 15) + "..." : session.title}</div>
                    <div className="text-xs text-workflow-text-muted">
                      {new Date(session.updatedAt).toLocaleDateString()}
                    </div>
                  </div>
                  <button
                    onClick={(e) => deleteSession(session.id, e)}
                    className="opacity-0 group-hover:opacity-100 p-1 hover:bg-workflow-surface-hover rounded transition-all"
                    title="Delete"
                  >
                    <Trash2 className="w-3 h-3 text-workflow-text-muted hover:text-rose-400" />
                  </button>
                </div>
              ))}
            </div>
          </ScrollArea>
        </div>
      )}

      {/* Main Chat Area */}
      <div className="flex-1 flex flex-col">
        {/* Toolbar */}
        <div className="px-4 pt-3 pb-1 flex justify-center">
          <div className="bg-workflow-surface border border-workflow-border rounded-xl px-4 h-14 flex items-center shadow-sm w-fit">
            <div className="flex items-center gap-6">
              <button
                type="button"
                onClick={() => router.push("/workflows")}
                className="flex items-center gap-1.5 text-workflow-text-muted hover:text-workflow-text transition-colors duration-200 shrink-0"
                title="Back to Workflows"
              >
                <ArrowLeft className="w-4 h-4" />
                <span className="font-mono text-xs">Back</span>
              </button>

              <div className="h-5 w-px bg-workflow-border shrink-0" />

              <div className="flex items-center gap-2 shrink-0">
                <Workflow className="w-4 h-4 text-workflow-text-muted" />
                <span className="text-workflow-text font-mono text-sm font-medium truncate max-w-48">
                  {workflowName || "Chat"}
                </span>
              </div>

              <div className="h-5 w-px bg-workflow-border shrink-0" />

              <div className="flex items-center gap-2">
                <div className="flex items-center bg-workflow-surface border border-workflow-border rounded-lg overflow-hidden">
                  <button
                    type="button"
                    onClick={() => router.push(`/workflows/${workflowId}`)}
                    title="Switch to workflow editor"
                    className="p-2.5 text-workflow-text-muted hover:text-workflow-text hover:bg-workflow-surface-hover transition-colors"
                  >
                    <Workflow className="w-4 h-4" />
                  </button>
                </div>

                <div className="flex items-center bg-workflow-surface border border-workflow-border rounded-lg overflow-hidden">
                  <button
                    type="button"
                    onClick={() => setShowSessionList(!showSessionList)}
                    title={showSessionList ? "Hide session list" : "Show session list"}
                    className="p-2.5 text-workflow-text-muted hover:text-workflow-text hover:bg-workflow-surface-hover transition-colors"
                  >
                    <MessageSquare className="w-4 h-4" />
                  </button>
                </div>

                <div className="flex items-center bg-workflow-surface border border-workflow-border rounded-lg overflow-hidden">
                  <button
                    type="button"
                    onClick={toggleNav}
                    title={navHidden ? "Show navigation" : "Hide navigation"}
                    className="p-2.5 text-workflow-text-muted hover:text-workflow-text hover:bg-workflow-surface-hover transition-colors"
                  >
                    {navHidden ? <PanelTop className="w-4 h-4" /> : <PanelTopClose className="w-4 h-4" />}
                  </button>
                </div>
              </div>
            </div>
          </div>
        </div>

        {/* Messages */}
        <ScrollArea className="flex-1 px-6 py-4">
          <div className="max-w-3xl mx-auto space-y-4">
            {messages.length === 0 && (
              <div className="text-center text-workflow-text-muted py-12">
                <Sparkles className="w-12 h-12 mx-auto mb-4 opacity-50" />
                <p className="text-lg font-medium mb-2">Start a conversation</p>
                <p className="text-sm">Type your message below and press Enter to send.</p>
              </div>
            )}
            
            {messages.map((message) => (
              <div key={message.id} className="space-y-3">
                {message.role === "user" && (
                  <div className="flex justify-end">
                    <div className="max-w-[80%] rounded-lg px-4 py-2 bg-primary text-primary-foreground">
                      <pre className="whitespace-pre-wrap font-sans text-sm">
                        {message.content}
                      </pre>
                    </div>
                  </div>
                )}
                
                {message.role === "assistant" && (
                  <>
                    {message.thinking && message.thinking.length > 0 && (
                      <div className="space-y-1">
                        <button
                          onClick={() => toggleThinking(message.id)}
                          className="flex items-center gap-2 text-xs text-workflow-text-muted hover:text-workflow-text transition-colors"
                        >
                          {collapsedThinking.has(message.id) ? (
                            <ChevronRight className="w-3 h-3" />
                          ) : (
                            <ChevronDown className="w-3 h-3" />
                          )}
                          <Brain className="w-3 h-3" />
                          <span>Thinking ({message.thinking.length} steps)</span>
                        </button>
                        
                        <div className={cn(
                          "ml-5 space-y-2 overflow-hidden transition-all",
                          collapsedThinking.has(message.id) ? "max-h-0" : "max-h-[2000px]"
                        )}>
                          {message.thinking.map((step, idx) => (
                            <div key={step.nodeId || idx} className="rounded-lg px-3 py-2 bg-workflow-surface border border-workflow-border">
                              <div className="flex items-center justify-between text-xs text-workflow-text-muted mb-1">
                                <div className="flex items-center gap-2">
                                  <Brain className="w-3 h-3" />
                                  <span className="font-medium">{step.nodeType || step.nodeId}</span>
                                </div>
                                {step.elapsed_time && (
                                  <span className="text-xs">{step.elapsed_time}</span>
                                )}
                              </div>
                              <pre className="whitespace-pre-wrap font-mono text-xs text-workflow-text-muted">
                                {step.content}
                              </pre>
                            </div>
                          ))}
                        </div>
                      </div>
                    )}
                    
                    {(message.content || isLoading) && (
                      <div className="flex justify-start">
                        <div className="max-w-[80%] rounded-lg px-4 py-2 bg-workflow-surface border border-workflow-border">
                          <div className="flex items-center gap-2 text-xs text-primary mb-1">
                            <MessageSquare className="w-3 h-3" />
                            <span>Reply</span>
                          </div>
                          {message.content ? (
                            <pre className="whitespace-pre-wrap font-sans text-sm text-workflow-text">
                              {message.content}
                            </pre>
                          ) : isLoading ? (
                            <Loader2 className="w-4 h-4 animate-spin text-workflow-text-muted" />
                          ) : null}
                        </div>
                      </div>
                    )}
                  </>
                )}
              </div>
            ))}
            
            <div ref={messagesEndRef} />
          </div>
        </ScrollArea>

        {/* Input */}
        <div className="px-6 py-4">
          <div className="max-w-3xl mx-auto flex gap-4">
            <Textarea
              value={input}
              onChange={(e) => setInput(e.target.value)}

              placeholder="Type your message..."
              className="min-h-[60px] resize-none bg-workflow-surface border-workflow-border text-workflow-text placeholder:text-workflow-text-muted"
              disabled={isLoading}
            />
            {isLoading ? (
              <button
                onClick={handleStop}
                disabled={isStopping}
                className="h-[60px] px-6 bg-rose-600 hover:bg-rose-500 text-white rounded-lg font-medium transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
              >
                {isStopping ? (
                  <Loader2 className="w-4 h-4 animate-spin" />
                ) : (
                  <Square className="w-4 h-4" />
                )}
              </button>
            ) : (
              <button
                onClick={handleSend}
                disabled={!input.trim()}
                className="h-[60px] px-6 bg-primary hover:bg-primary/90 text-primary-foreground rounded-lg font-medium transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
              >
                <Send className="w-4 h-4" />
              </button>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
