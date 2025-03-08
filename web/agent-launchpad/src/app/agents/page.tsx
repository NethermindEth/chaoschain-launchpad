"use client";

import { useState } from "react";
import Head from "next/head";

interface Agent {
  id: string;
  name: string;
  role: 'producer' | 'validator';
  traits: string[];
  style: string;
  influences: string[];
  mood: string;
  api_key: string;
  endpoint: string;
}

export default function StartChainPage() {
  const [agents, setAgents] = useState<Agent[]>([]);
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [formData, setFormData] = useState({
    name: "",
    role: "producer",
    traits: "",
    style: "",
    influences: "",
    mood: "",
    api_key: "",
    endpoint: "",
  });

  const handleChange = (
    e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>
  ) => {
    setFormData({ ...formData, [e.target.name]: e.target.value });
  };

  const handleSubmit = (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();

    // Create a new Agent using the form data. For traits and influences, assume a comma-separated list.
    const newAgent: Agent = {
      id: Date.now().toString(),
      name: formData.name,
      role: formData.role as "producer" | "validator",
      traits: formData.traits.split(",").map(str => str.trim()).filter(Boolean),
      style: formData.style,
      influences: formData.influences.split(",").map(str => str.trim()).filter(Boolean),
      mood: formData.mood,
      api_key: formData.api_key,
      endpoint: formData.endpoint,
    };

    setAgents([...agents, newAgent]);

    // Reset the form and close the modal
    setFormData({
      name: "",
      role: "producer",
      traits: "",
      style: "",
      influences: "",
      mood: "",
      api_key: "",
      endpoint: "",
    });
    setIsModalOpen(false);
  };

  return (
    <>
      <Head>
        <title>Start Chain - ChaosChain Agent Launchpad</title>
        <meta name="viewport" content="width=device-width, initial-scale=1" />
      </Head>
      <div className={`min-h-screen bg-gray-900 text-gray-100 p-8`}>
        <h1 className="text-4xl font-bold mb-6">Start Chain</h1>
        <button
          className="bg-gradient-to-r from-purple-700 to-purple-900 hover:opacity-90 text-gray-100 font-medium py-3 px-6 rounded-lg transition-all duration-300 mb-6"
          onClick={() => setIsModalOpen(true)}
        >
          Add Agent
        </button>

        <div>
          <h2 className="text-2xl font-bold mb-4">Agents Added</h2>
          {agents.length === 0 ? (
            <p>No agents added yet.</p>
          ) : (
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              {agents.map((agent) => (
                <div key={agent.id} className="bg-gray-800 p-4 rounded-lg">
                  <h3 className="text-xl font-semibold mb-2">{agent.name}</h3>
                  <p><strong>Role:</strong> {agent.role}</p>
                  <p><strong>Traits:</strong> {agent.traits.join(", ")}</p>
                  <p><strong>Style:</strong> {agent.style}</p>
                  <p><strong>Influences:</strong> {agent.influences.join(", ")}</p>
                  <p><strong>Mood:</strong> {agent.mood}</p>
                  <p><strong>API Key:</strong> {agent.api_key}</p>
                  <p><strong>Endpoint:</strong> {agent.endpoint}</p>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Modal for Adding an Agent */}
        {isModalOpen && (
          <div className="fixed inset-0 flex items-center justify-center bg-black bg-opacity-50 z-50">
            <div className="bg-gray-800 p-6 rounded-lg w-full max-w-lg">
              <div className="flex justify-between items-center mb-4">
                <h2 className="text-2xl font-bold">Add New Agent</h2>
                <button
                  onClick={() => setIsModalOpen(false)}
                  className="text-gray-300 hover:text-gray-100"
                >
                  X
                </button>
              </div>
              <form onSubmit={handleSubmit} className="space-y-4">
                <div>
                  <label htmlFor="name" className="block mb-1">
                    Name
                  </label>
                  <input
                    type="text"
                    name="name"
                    id="name"
                    value={formData.name}
                    onChange={handleChange}
                    required
                    className="w-full p-2 rounded bg-gray-700 text-gray-100"
                  />
                </div>
                <div>
                  <label htmlFor="role" className="block mb-1">
                    Role
                  </label>
                  <select
                    id="role"
                    name="role"
                    value={formData.role}
                    onChange={handleChange}
                    className="w-full p-2 rounded bg-gray-700 text-gray-100"
                  >
                    <option value="producer">Producer</option>
                    <option value="validator">Validator</option>
                  </select>
                </div>
                <div>
                  <label htmlFor="traits" className="block mb-1">
                    Traits (comma separated)
                  </label>
                  <input
                    type="text"
                    name="traits"
                    id="traits"
                    value={formData.traits}
                    onChange={handleChange}
                    className="w-full p-2 rounded bg-gray-700 text-gray-100"
                  />
                </div>
                <div>
                  <label htmlFor="style" className="block mb-1">
                    Style
                  </label>
                  <input
                    type="text"
                    name="style"
                    id="style"
                    value={formData.style}
                    onChange={handleChange}
                    className="w-full p-2 rounded bg-gray-700 text-gray-100"
                  />
                </div>
                <div>
                  <label htmlFor="influences" className="block mb-1">
                    Influences (comma separated)
                  </label>
                  <input
                    type="text"
                    name="influences"
                    id="influences"
                    value={formData.influences}
                    onChange={handleChange}
                    className="w-full p-2 rounded bg-gray-700 text-gray-100"
                  />
                </div>
                <div>
                  <label htmlFor="mood" className="block mb-1">
                    Mood
                  </label>
                  <input
                    type="text"
                    name="mood"
                    id="mood"
                    value={formData.mood}
                    onChange={handleChange}
                    className="w-full p-2 rounded bg-gray-700 text-gray-100"
                  />
                </div>
                
                <div className="flex justify-end space-x-4">
                  <button
                    type="button"
                    onClick={() => setIsModalOpen(false)}
                    className="bg-gray-600 hover:bg-gray-500 text-gray-100 py-2 px-4 rounded"
                  >
                    Cancel
                  </button>
                  <button
                    type="submit"
                    className="bg-gradient-to-r from-purple-700 to-purple-900 hover:opacity-90 text-gray-100 font-medium py-2 px-4 rounded transition-all duration-300"
                  >
                    Save Agent
                  </button>
                </div>
              </form>
            </div>
          </div>
        )}
      </div>
    </>
  );
}