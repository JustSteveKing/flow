import * as React from "react";
import { cn } from "@/lib/utils";

const base =
  "inline-flex items-center rounded-full border px-2 py-0.5 text-xs font-medium transition-colors";

// tone maps a flow status/appetite/layer to a colour. Unknown tones fall back to
// a neutral chip.
const tones: Record<string, string> = {
  neutral: "border-transparent bg-secondary text-muted-foreground",
  // pitch / spec / adr / cycle statuses
  shaping: "border-transparent bg-[hsl(43_40%_20%)] text-uphill",
  specifying: "border-transparent bg-[hsl(43_40%_20%)] text-uphill",
  proposed: "border-transparent bg-[hsl(43_40%_20%)] text-uphill",
  bet: "border-transparent bg-accent text-accent-foreground",
  building: "border-transparent bg-accent text-accent-foreground",
  accepted: "border-transparent bg-accent text-accent-foreground",
  active: "border-transparent bg-accent text-accent-foreground",
  done: "border-transparent bg-[hsl(212_50%_22%)] text-done",
  frozen: "border-transparent bg-[hsl(212_50%_22%)] text-done",
  closed: "border-transparent bg-[hsl(212_50%_22%)] text-done",
  shelved: "border-transparent bg-secondary text-muted-foreground",
  superseded: "border-transparent bg-secondary text-muted-foreground",
  cooldown: "border-transparent bg-[hsl(140_25%_20%)] text-downhill",
  downhill: "border-transparent bg-[hsl(140_25%_20%)] text-downhill",
  uphill: "border-transparent bg-[hsl(43_40%_20%)] text-uphill",
  // appetites
  "big-batch": "border-transparent bg-accent text-accent-foreground",
  "small-batch": "border-transparent bg-[hsl(140_25%_20%)] text-downhill",
  // layers
  baseline: "border-transparent bg-[hsl(262_35%_28%)] text-[hsl(262_85%_82%)]",
  local: "border-transparent bg-[hsl(140_25%_20%)] text-downhill",
};

export function Badge({
  tone = "neutral",
  className,
  ...props
}: React.HTMLAttributes<HTMLSpanElement> & { tone?: string }) {
  return <span className={cn(base, tones[tone] ?? tones.neutral, className)} {...props} />;
}
