import type { HillPoint } from "../api";
import { useUI } from "../ui";
import { groupBy } from "@/lib/group";
import { Card } from "@/components/ui/card";
import { HillTrack } from "./HillTrack";

export function PortfolioHill({ points }: { points: HillPoint[] }) {
  const { openDoc } = useUI();

  if (points.length === 0) {
    return <Card className="p-4 text-sm italic text-muted-foreground">nothing building</Card>;
  }

  return (
    <Card className="p-4">
      {groupBy(points, (p) => p.projectId).map(([pid, group]) => (
        <div key={pid} className="mb-3 last:mb-0">
          <div className="mb-1 font-semibold text-primary">{pid}</div>
          {group.map((p) => (
            <div
              key={p.lineage + p.scopeId}
              onClick={() => openDoc({ project: p.projectId, kind: "spec", id: p.lineage, label: p.lineage })}
              className="grid cursor-pointer grid-cols-[240px_1fr] items-center gap-3 rounded-md py-1.5 hover:bg-secondary/50"
              title="view source"
            >
              <div className="text-sm">
                {p.title || p.scopeId}{" "}
                <span className="font-mono text-xs text-muted-foreground">{p.scopeId}</span>
              </div>
              <HillTrack hill={p.hill} status={p.status} />
            </div>
          ))}
        </div>
      ))}
    </Card>
  );
}
