import { cn } from "@/lib/utils";

export function Toast({ message }: { message: string | null }) {
  return (
    <div
      className={cn(
        "fixed bottom-5 right-5 max-w-sm rounded-lg border border-destructive/50 bg-destructive/20 px-4 py-2.5 text-sm text-destructive-foreground transition-all",
        message ? "translate-y-0 opacity-100" : "pointer-events-none translate-y-2 opacity-0"
      )}
      role="status"
    >
      {message}
    </div>
  );
}
