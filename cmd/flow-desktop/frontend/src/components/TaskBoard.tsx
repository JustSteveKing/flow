import { useState } from "react";
import type { TaskCard } from "../api";
import { api } from "../api";
import { useUI } from "../ui";
import { Badge } from "@/components/ui/badge";
import { Card } from "@/components/ui/card";
import { cn } from "@/lib/utils";

// Fixed column order, mirroring the generated TASKS.md board: active work first,
// terminal last.
const COLUMNS: string[] = ["todo", "doing", "blocked", "review", "done", "dropped"];

// short strips the agent:/human: prefix for the compact @name on a card.
function short(assignee: string): string {
  const i = assignee.indexOf(":");
  return i >= 0 ? assignee.slice(i + 1) : assignee;
}

export function TaskBoard({ tasks }: { tasks: TaskCard[] }) {
  const { toast, refresh } = useUI();
  const [dragId, setDragId] = useState<string | null>(null);
  const [over, setOver] = useState<string | null>(null);

  if (tasks.length === 0) {
    return <Card className="p-4 text-sm italic text-muted-foreground">no tasks across the plane</Card>;
  }

  const byStatus = (status: string) => tasks.filter((t) => t.status === status);

  async function drop(status: string, t: TaskCard) {
    setOver(null);
    setDragId(null);
    if (t.status === status) return; // no-op drop onto the same column
    try {
      await api.taskStatus(t.projectId, t.id, status);
      refresh();
    } catch (e) {
      // The state machine rejected the drop; the card stays where it was.
      toast(String((e as Error).message));
    }
  }

  return (
    <div className="flex gap-3 overflow-x-auto pb-2">
      {COLUMNS.map((status) => {
        const cards = byStatus(status);
        return (
          <div
            key={status}
            onDragOver={(e) => {
              e.preventDefault();
              setOver(status);
            }}
            onDragLeave={() => setOver((cur) => (cur === status ? null : cur))}
            onDrop={() => {
              const t = tasks.find((x) => x.id === dragId);
              if (t) void drop(status, t);
            }}
            className={cn(
              "flex w-64 shrink-0 flex-col rounded-lg border bg-card/40 p-2",
              over === status && "border-primary bg-secondary/40"
            )}
          >
            <div className="mb-2 flex items-center justify-between px-1">
              <span className="text-xs font-semibold uppercase tracking-widest text-muted-foreground">
                {status}
              </span>
              <span className="text-xs text-muted-foreground">{cards.length}</span>
            </div>

            <div className="flex flex-col gap-2">
              {cards.map((t) => (
                <Card
                  key={t.projectId + t.id}
                  draggable
                  onDragStart={() => setDragId(t.id)}
                  onDragEnd={() => {
                    setDragId(null);
                    setOver(null);
                  }}
                  className={cn(
                    "cursor-grab p-2.5 active:cursor-grabbing",
                    dragId === t.id && "opacity-50"
                  )}
                  title={t.projectId}
                >
                  <div className="mb-1 flex items-center gap-2">
                    <span className="font-mono text-xs text-muted-foreground">{t.id}</span>
                    {t.total > 0 && (
                      <span className="font-mono text-xs text-muted-foreground">
                        {t.done}/{t.total}
                      </span>
                    )}
                  </div>
                  <div className="mb-1 text-sm leading-snug">{t.title || "(untitled)"}</div>
                  <div className="flex flex-wrap items-center gap-1">
                    {t.assignee && <Badge tone="active">@{short(t.assignee)}</Badge>}
                    {(t.blockers?.length ?? 0) > 0 && (
                      <Badge tone="shelved" title={t.blockers!.join(", ")}>
                        blocked by {t.blockers!.length}
                      </Badge>
                    )}
                    {(t.tags ?? []).map((tag) => (
                      <Badge key={tag} tone="neutral">
                        {tag}
                      </Badge>
                    ))}
                  </div>
                </Card>
              ))}
            </div>
          </div>
        );
      })}
    </div>
  );
}
