import type { BetRow } from "../api";
import { api } from "../api";
import { useUI } from "../ui";
import { groupBy } from "@/lib/group";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { cn } from "@/lib/utils";

export function BettingTable({ rows }: { rows: BetRow[] }) {
  const { openDoc, toast, refresh } = useUI();

  if (rows.length === 0) {
    return <Card className="p-4 text-sm italic text-muted-foreground">no active bets across the plane</Card>;
  }

  async function shelve(r: BetRow) {
    try {
      await api.shelve(r.projectId, r.lineage);
      refresh();
    } catch (e) {
      toast(String((e as Error).message));
    }
  }

  return (
    <Card className="p-4">
      {groupBy(rows, (r) => r.projectId).map(([pid, group]) => (
        <div key={pid} className="mb-3 last:mb-0">
          <div className="mb-1 font-semibold text-primary">{pid}</div>
          {group.map((r) => (
            <div
              key={r.lineage}
              onClick={() => openDoc({ project: r.projectId, kind: "pitch", id: r.lineage, label: r.lineage })}
              className={cn(
                "flex cursor-pointer items-center gap-3 rounded-md border-b border-border px-2 py-1.5 last:border-0 hover:bg-secondary/50",
                r.shelved && "opacity-50"
              )}
              title="view source"
            >
              <span className="min-w-56 font-mono text-xs text-muted-foreground">{r.lineage}</span>
              <span className="flex-1">{r.title || "(untitled)"}</span>
              {r.shelved ? (
                <Badge tone="shelved">shelved</Badge>
              ) : (
                <>
                  <Badge tone={r.appetite}>{r.appetite}</Badge>
                  <Button
                    size="sm"
                    variant="outline"
                    onClick={(e) => {
                      e.stopPropagation();
                      shelve(r);
                    }}
                  >
                    shelve
                  </Button>
                </>
              )}
            </div>
          ))}
        </div>
      ))}
    </Card>
  );
}
