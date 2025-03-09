import { useState } from "react";

// Shared Agent interface
export interface Agent {
  id: string;
  name: string;
  role: "producer" | "validator";
  traits: string[];
  style: string;
  influences: string[];
  mood: string;
}

export function useAgentsManager() {
  const [agents, setAgents] = useState<Agent[]>([]);
  const [isModalOpen, setIsModalOpen] = useState(false);

  const addAgent = (agent: Agent) => {
    setAgents((prev) => [...prev, agent]);
  };

  const openModal = () => setIsModalOpen(true);
  const closeModal = () => setIsModalOpen(false);

  const requiredAgents = 3;
  const progressPercentage = Math.min((agents.length / requiredAgents) * 100, 100);

  return {
    agents,
    isModalOpen,
    addAgent,
    openModal,
    closeModal,
    requiredAgents,
    progressPercentage,
  };
} 