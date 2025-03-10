import { useState } from "react";
import type { Agent } from "./useAgentsManager";

export function useAgentForm() {
  const [formData, setFormData] = useState({
    name: "",
    role: "validator",
    style: "",
    mood: "",
  });
  const [traits, setTraits] = useState<string[]>([]);
  const [influences, setInfluences] = useState<string[]>([]);
  const [newTrait, setNewTrait] = useState("");
  const [newInfluence, setNewInfluence] = useState("");

  const handleChange = (
    e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>
  ) => {
    setFormData({ ...formData, [e.target.name]: e.target.value });
  };

  const addTrait = () => {
    if (newTrait.trim() !== "") {
      setTraits([...traits, newTrait.trim()]);
      setNewTrait("");
    }
  };

  const removeTrait = (index: number) => {
    setTraits((prev) => prev.filter((_, i) => i !== index));
  };

  const addInfluence = () => {
    if (newInfluence.trim() !== "") {
      setInfluences([...influences, newInfluence.trim()]);
      setNewInfluence("");
    }
  };

  const removeInfluence = (index: number) => {
    setInfluences((prev) => prev.filter((_, i) => i !== index));
  };

  const resetForm = () => {
    setFormData({
      name: "",
      role: "producer",
      style: "",
      mood: "",
    });
    setTraits([]);
    setInfluences([]);
    setNewTrait("");
    setNewInfluence("");
  };

  const buildAgent = (): Agent => ({
    id: Date.now().toString(),
    name: formData.name,
    role: formData.role as "producer" | "validator",
    traits,
    style: formData.style,
    influences,
    mood: formData.mood,
  });

  return {
    formData,
    setFormData,
    traits,
    influences,
    newTrait,
    newInfluence,
    setNewTrait,        // Exposing setter for newTrait
    setNewInfluence,    // Exposing setter for newInfluence
    handleChange,
    addTrait,
    removeTrait,
    addInfluence,
    removeInfluence,
    resetForm,
    buildAgent,
  };
}