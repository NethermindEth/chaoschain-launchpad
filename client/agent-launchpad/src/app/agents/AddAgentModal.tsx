"use client";

import { FiPlus, FiTrash, FiX, FiCheck, FiUserPlus } from "react-icons/fi";
import { useAgentForm } from "./hooks/useAgentForm";
import type { Agent } from "./hooks/useAgentsManager";

interface AddAgentModalProps {
  onAddAgent: (agent: Agent) => void;
  onClose: () => void;
}

export default function AddAgentModal({
  onAddAgent,
  onClose,
}: AddAgentModalProps) {
  const {
    formData,
    handleChange,
    traits,
    influences,
    newTrait,
    newInfluence,
    setNewTrait,     // Use the setter here
    setNewInfluence, // Use the setter here
    addTrait,
    removeTrait,
    addInfluence,
    removeInfluence,
    resetForm,
    buildAgent,
  } = useAgentForm();

  // Form submission: build the agent and pass it to the parent
  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      const agent = buildAgent();
      await onAddAgent(agent);
      resetForm();
      onClose();
    } catch (error) {
      console.error('Failed to add agent:', error);
      // You might want to show an error message to the user here
    }
  };

  return (
    <div className="fixed inset-0 flex items-center justify-center bg-black bg-opacity-50 z-50 transition-opacity duration-300">
      <div className="bg-gray-800 p-6 rounded-lg w-full max-w-lg overflow-y-auto max-h-full shadow-xl transform transition-all duration-300">
        <div className="flex justify-between items-center mb-4">
          <h2 className="text-2xl font-bold flex items-center text-white">
            <FiUserPlus className="mr-2 text-green-400" /> Add New Agent
          </h2>
          <button
            onClick={onClose}
            className="text-gray-300 hover:text-gray-100 transition-colors"
          >
            <FiX size={24} />
          </button>
        </div>
        <form onSubmit={handleSubmit} className="space-y-4">
          {/* Other inputs */}
          <div>
            <label htmlFor="name" className="block mb-1 font-medium text-gray-300">
              Name
            </label>
            <input
              type="text"
              name="name"
              id="name"
              value={formData.name}
              onChange={handleChange}
              required
              placeholder="e.g., Agent Alpha"
              className="w-full p-2 rounded bg-gray-700 text-gray-100 placeholder-gray-400"
            />
          </div>
          <div>
            <label htmlFor="role" className="block mb-1 font-medium text-gray-300">
              Role
            </label>
            <select
              id="role"
              name="role"
              value={formData.role}
              onChange={handleChange}
              className="w-full p-2 rounded bg-gray-700 text-gray-100"
            >
              <option value="validator">Validator</option>
              <option value="producer">Producer</option>
            </select>
          </div>
          {/* Traits Section */}
          <div>
            <label className="block mb-1 font-medium text-gray-300">Traits</label>
            <div className="space-y-2">
              {traits.map((trait, index) => (
                <div
                  key={index}
                  className="flex items-center justify-between bg-gray-700 p-2 rounded text-gray-100"
                >
                  <span>{trait}</span>
                  <button
                    type="button"
                    onClick={() => removeTrait(index)}
                    className="text-red-500 hover:text-red-400 transition-colors"
                  >
                    <FiTrash />
                  </button>
                </div>
              ))}
              <div className="flex items-center space-x-2">
                <input
                  type="text"
                  value={newTrait}
                  onChange={(e) => setNewTrait(e.target.value)}
                  placeholder="e.g., Innovative"
                  className="w-full p-2 rounded bg-gray-700 text-gray-100 placeholder-gray-400"
                />
                <button
                  type="button"
                  onClick={addTrait}
                  className="flex items-center bg-green-600 hover:bg-green-500 text-gray-100 py-2 px-3 rounded transition-transform transform hover:scale-105"
                >
                  <FiPlus />
                </button>
              </div>
            </div>
          </div>
          {/* Style Field */}
          <div>
            <label htmlFor="style" className="block mb-1 font-medium text-gray-300">
              Style
            </label>
            <input
              type="text"
              name="style"
              id="style"
              value={formData.style}
              onChange={handleChange}
              placeholder="e.g., Cyberpunk"
              className="w-full p-2 rounded bg-gray-700 text-gray-100 placeholder-gray-400"
            />
          </div>
          {/* Influences Section */}
          <div>
            <label className="block mb-1 font-medium text-gray-300">Influences</label>
            <div className="space-y-2">
              {influences.map((influence, index) => (
                <div
                  key={index}
                  className="flex items-center justify-between bg-gray-700 p-2 rounded text-gray-100"
                >
                  <span>{influence}</span>
                  <button
                    type="button"
                    onClick={() => removeInfluence(index)}
                    className="text-red-500 hover:text-red-400 transition-colors"
                  >
                    <FiTrash />
                  </button>
                </div>
              ))}
              <div className="flex items-center space-x-2">
                <input
                  type="text"
                  value={newInfluence}
                  onChange={(e) => setNewInfluence(e.target.value)}
                  placeholder="e.g., Elon Musk"
                  className="w-full p-2 rounded bg-gray-700 text-gray-100 placeholder-gray-400"
                />
                <button
                  type="button"
                  onClick={addInfluence}
                  className="flex items-center bg-green-600 hover:bg-green-500 text-gray-100 py-2 px-3 rounded transition-transform transform hover:scale-105"
                >
                  <FiPlus />
                </button>
              </div>
            </div>
          </div>
          <div>
            <label htmlFor="mood" className="block mb-1 font-medium text-gray-300">
              Mood
            </label>
            <input
              type="text"
              name="mood"
              id="mood"
              value={formData.mood}
              onChange={handleChange}
              placeholder="e.g., Optimistic"
              className="w-full p-2 rounded bg-gray-700 text-gray-100 placeholder-gray-400"
            />
          </div>
          <div className="flex justify-end space-x-4">
            <button
              type="button"
              onClick={onClose}
              className="flex items-center bg-gray-600 hover:bg-gray-500 text-gray-100 py-2 px-4 rounded transition-transform transform hover:scale-105"
            >
              <FiX className="mr-1" /> Cancel
            </button>
            <button
              type="submit"
              className="flex items-center bg-gradient-to-r from-purple-700 to-purple-900 hover:opacity-90 text-gray-100 font-medium py-2 px-4 rounded transition transform hover:scale-105 duration-200"
            >
              <FiCheck className="mr-1" /> Save Agent
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}