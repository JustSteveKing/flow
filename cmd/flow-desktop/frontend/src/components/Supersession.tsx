import { ArrowRight } from "lucide-react";
import type { Edge } from "../api";
import { useUI } from "../ui";
import { groupBy } from "@/lib/group";
import { Badge } from "@/components/ui/badge";
import { Card } from "@/components/ui/card";

export function Supersession({ edges }: { edges: Edge[] }) {
  const { openDoc } = useUI();

  if (edges.length === 0) {
    return <Card className="p-4 text-sm italic text-muted-foreground">no supersessions</Card>;
  }

  return (
    <Card className="p-4">
      {groupBy(edges, (e) => e.projectId).map(([pid, group]) => (
        <div key={pid} className="mb-3 last:mb-0">
          <div className="mb-1 font-semibold text-primary">{pid}</div>
          {group.map((e) => (
            <div key={e.from} className="flex items-center gap-2 py-1.5 font-mono text-sm">
              <Badge
                tone={e.fromLayer}
                className="cursor-pointer"
                title="view source"
                onClick={() => openDoc({ project: e.projectId, kind: "decision", id: e.from, label: e.from })}
              >
                {e.from}
              </Badge>
              <ArrowRight className="h-4 w-4 text-muted-foreground" />
              <Badge
                tone={e.toLayer}
                className="cursor-pointer"
                title="view source"
                onClick={() => openDoc({ project: e.projectId, kind: "decision", id: e.to, label: e.to })}
              >
                {e.to}
              </Badge>
            </div>
          ))}
        </div>
      ))}
    </Card>
  );
}
