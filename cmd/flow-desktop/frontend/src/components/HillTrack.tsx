import { cn } from "@/lib/utils";

const markerColour: Record<string, string> = {
  uphill: "bg-uphill",
  downhill: "bg-downhill",
  done: "bg-done",
};

// HillTrack renders a Shape Up hill: a crest at the midpoint and a marker at the
// scope's position. size "sm" is used inside detail cards.
export function HillTrack({
  hill,
  status,
  size = "md",
}: {
  hill: number;
  status: string;
  size?: "sm" | "md";
}) {
  const h = size === "sm" ? "h-2" : "h-6";
  const dot = size === "sm" ? "h-3 w-3 -top-0.5" : "h-5 w-5 top-0.5";
  return (
    <div className={cn("relative w-full rounded-full bg-secondary/70", h, size === "sm" && "max-w-40")}>
      <div className="absolute left-1/2 top-0 bottom-0 w-px bg-border" />
      <div
        className={cn("absolute -translate-x-1/2 rounded-full", dot, markerColour[status] ?? "bg-muted-foreground")}
        style={{ left: `${Math.max(0, Math.min(1, hill)) * 100}%` }}
        title={`${status} ${hill.toFixed(2)}`}
      />
    </div>
  );
}
