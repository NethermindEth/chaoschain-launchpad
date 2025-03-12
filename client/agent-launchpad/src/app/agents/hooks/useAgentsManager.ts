import { useState } from "react";
import { registerAgent, ApiError } from "@/services/api";

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
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const addAgent = async (agent: Agent) => {
    setIsLoading(true);
    setError(null);
    try {
      const response = await registerAgent(agent);
      setAgents(prev => [...prev, { ...agent, id: response.agentID }]);
      return response;
    } catch (err) {
      const errorMessage = err instanceof ApiError 
        ? err.message 
        : 'Failed to register agent';
      setError(errorMessage);
      throw err;
    } finally {
      setIsLoading(false);
    }
  };

  const openModal = () => setIsModalOpen(true);
  const closeModal = () => setIsModalOpen(false);

  const requiredAgents = 3;
  const progressPercentage = Math.min((agents.length / requiredAgents) * 100, 100);

  return {
    agents,
    isModalOpen,
    isLoading,
    error,
    addAgent,
    openModal,
    closeModal,
    requiredAgents,
    progressPercentage,
  };
} 