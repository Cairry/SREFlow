import { WorkflowEditor } from "@/components/workflow/workflow-editor";

export default function WorkflowPage({ params }: { params: Promise<{ id: string }> }) {
  return (
    <div className="h-[calc(100vh-64px)]">
      <WorkflowEditor workflowId={params} />
    </div>
  );
}
