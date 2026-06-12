import { WorkflowEditor } from "@/components/workflow/workflow-editor";

export default function WorkflowPage({ params }: { params: Promise<{ id: string }> }) {
  return (
    <div className="h-screen">
      <WorkflowEditor workflowId={params} />
    </div>
  );
}
