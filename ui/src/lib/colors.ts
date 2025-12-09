
// Stable hash function for string -> integer
function simpleHash(str: string): number {
    let hash = 0;
    for (let i = 0; i < str.length; i++) {
        const char = str.charCodeAt(i);
        hash = (hash << 5) - hash + char;
        hash |= 0; // Convert to 32bit integer
    }
    // Ensure positive
    return Math.abs(hash);
}

// Predefined palette for better aesthetics (Tailwind colors)
// Using distinct, readable colors (skipped yellows/light grays)
const PALETTE = [
    "blue",
    "purple",
    "emerald",
    "indigo",
    "pink",
    "teal",
    "cyan",
    "orange",
    "rose",
    "violet"
] as const;

export type AgentColor = typeof PALETTE[number];

export function getAgentColor(name: string): AgentColor {
    if (!name) return "blue";

    // Clean name (case-insensitive)
    const normalized = name.trim().toLowerCase();

    // Specific overrides for known system roles if desired
    if (normalized === "assistant" || normalized === "hector") return "blue";

    const hash = simpleHash(normalized);
    const index = hash % PALETTE.length;

    return PALETTE[index];
}

// Helper to get Tailwind classes based on color
export function getAgentColorClasses(color: AgentColor) {
    const map: Record<AgentColor, { bg: string, text: string, border: string, ring: string }> = {
        blue: { bg: "bg-blue-600", text: "text-blue-500", border: "border-blue-500/30", ring: "ring-blue-500/20" },
        purple: { bg: "bg-purple-600", text: "text-purple-500", border: "border-purple-500/30", ring: "ring-purple-500/20" },
        emerald: { bg: "bg-emerald-600", text: "text-emerald-500", border: "border-emerald-500/30", ring: "ring-emerald-500/20" },
        indigo: { bg: "bg-indigo-600", text: "text-indigo-500", border: "border-indigo-500/30", ring: "ring-indigo-500/20" },
        pink: { bg: "bg-pink-600", text: "text-pink-500", border: "border-pink-500/30", ring: "ring-pink-500/20" },
        teal: { bg: "bg-teal-600", text: "text-teal-500", border: "border-teal-500/30", ring: "ring-teal-500/20" },
        cyan: { bg: "bg-cyan-600", text: "text-cyan-500", border: "border-cyan-500/30", ring: "ring-cyan-500/20" },
        orange: { bg: "bg-orange-600", text: "text-orange-500", border: "border-orange-500/30", ring: "ring-orange-500/20" },
        rose: { bg: "bg-rose-600", text: "text-rose-500", border: "border-rose-500/30", ring: "ring-rose-500/20" },
        violet: { bg: "bg-violet-600", text: "text-violet-500", border: "border-violet-500/30", ring: "ring-violet-500/20" },
    };
    return map[color];
}
