import React, { useMemo } from "react";
import * as yaml from "js-yaml";
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuLabel,
    DropdownMenuTrigger,
} from "./ui/dropdown-menu";
import { Button } from "./ui/button";
import { ChevronDown, Check } from "lucide-react";
import { cn } from "../lib/utils";
import { getAgentBranding, AgentBranding } from "../lib/colors";
import { useStore } from "../store/useStore";
// Import Agent type to ensure we have access to 'url'
import type { Agent } from "../types";

interface AgentSelectProps {
    agents: Agent[];
    selectedAgent: Agent | null;
    onSelect: (agentName: string) => void;
    className?: string;
    variant?: "default" | "destructive" | "outline" | "secondary" | "ghost" | "link";
}

export const AgentSelect: React.FC<AgentSelectProps> = ({
    agents,
    selectedAgent,
    onSelect,
    className,
    variant = "outline",
}) => {
    // Access the current YAML content from the store to resolve agent types
    const studioYamlContent = useStore((state) => state.studioYamlContent);

    // Memoize the parsed config to avoid re-parsing on every render
    const appConfig = useMemo(() => {
        try {
            return (yaml.load(studioYamlContent) as any) || {};
        } catch (e) {
            return {};
        }
    }, [studioYamlContent]);

    // Helper to resolve branding including type from config
    const getBranding = (agent: Agent): AgentBranding => {
        // 1. Try to find Type by exact Name match (often works if Name == ID)
        let type = appConfig.agents?.[agent.name]?.type;

        // 2. If not found, try to extract ID from URL (/agents/{id})
        // This handles cases where Name in UI is different from ID in YAML
        if (!type && agent.url) {
            try {
                const cleanUrl = agent.url.endsWith('/') ? agent.url.slice(0, -1) : agent.url;
                const parts = cleanUrl.split('/');
                const id = parts[parts.length - 1];
                type = appConfig.agents?.[id]?.type;
            } catch (e) {
                // Ignore parsing errors
            }
        }

        // Use name only for color hashing - matches canvas AgentNode behavior
        return getAgentBranding({ name: agent.name, type });
    };

    const selectedBranding = selectedAgent ? getBranding(selectedAgent) : null;

    return (
        <DropdownMenu>
            <DropdownMenuTrigger asChild>
                <Button
                    variant={variant}
                    className={cn("w-full justify-between px-3", className)}
                >
                    <div className="flex items-center gap-2 truncate">
                        {selectedBranding ? (
                            <>
                                <div
                                    className={cn(
                                        "flex items-center justify-center w-5 h-5 rounded shadow-sm border",
                                        selectedBranding.classes.bg,
                                        selectedBranding.classes.border
                                    )}
                                >
                                    <selectedBranding.Icon size={12} className="text-white" />
                                </div>
                                <span className="truncate">{selectedAgent!.name}</span>
                            </>
                        ) : (
                            <span className="opacity-70">Select Agent</span>
                        )}
                    </div>
                    <ChevronDown size={14} className="opacity-50 ml-2 shrink-0" />
                </Button>
            </DropdownMenuTrigger>

            <DropdownMenuContent
                className="min-w-[220px] bg-[#1e1e1e]/95 backdrop-blur-xl rounded-xl border border-white/10 shadow-2xl p-1.5"
                sideOffset={5}
                align="start"
            >
                <DropdownMenuLabel className="px-2 py-1.5 text-xs font-semibold text-gray-500 uppercase tracking-wider mb-1">
                    Available Agents
                </DropdownMenuLabel>

                {agents.map((agent) => {
                    const branding = getBranding(agent);
                    const isSelected = selectedAgent?.name === agent.name;

                    return (
                        <DropdownMenuItem
                            key={agent.name}
                            className={cn(
                                "group flex items-center gap-3 px-2 py-2 rounded-lg text-sm cursor-pointer transition-colors relative pl-9",
                                isSelected
                                    ? "bg-white/10 text-white focus:bg-white/15 focus:text-white"
                                    : "text-gray-400 hover:text-white hover:bg-white/5 focus:bg-white/5 focus:text-white"
                            )}
                            onSelect={() => onSelect(agent.name)}
                        >
                            {/* Active Indicator */}
                            <div className="absolute left-2 flex items-center justify-center">
                                {isSelected && <Check size={14} className="text-white" />}
                            </div>

                            {/* Agent Icon with Dynamic Color */}
                            <div
                                className={cn(
                                    "w-6 h-6 rounded flex items-center justify-center transition-colors border shadow-inner",
                                    branding.classes.bg,
                                    branding.classes.border,
                                    isSelected
                                        ? "opacity-100"
                                        : "opacity-70 group-hover:opacity-100 group-focus:opacity-100"
                                )}
                            >
                                <branding.Icon size={14} className="text-white" />
                            </div>

                            <div className="flex flex-col">
                                <span className="font-medium leading-none">{agent.name}</span>
                                {branding.label && (
                                    <span className="text-[10px] uppercase tracking-wider opacity-60 mt-0.5">
                                        {branding.label}
                                    </span>
                                )}
                            </div>
                        </DropdownMenuItem>
                    );
                })}
            </DropdownMenuContent>
        </DropdownMenu>
    );
};
