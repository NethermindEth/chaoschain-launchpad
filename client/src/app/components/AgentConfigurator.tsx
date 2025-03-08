"use client";

import React, { useState } from "react";
import { Agent } from "@/types/agent";

const ELIZA_SERVER = "http://127.0.0.1:3002";

export default function AgentConfigurator() {
  const [agents, setAgents] = useState<Agent[]>([]);
  const [loading, setLoading] = useState(false);
  const [currentAgent, setCurrentAgent] = useState<Partial<Agent>>({
    name: "",
    bio: [],
    lore: [],
    style: { all: [], chat: [], post: [] },
    messageExamples: [],
    plugins: [],
    clients: [],
    adjectives: [],
    topics: [],
    postExamples: [],
    modelProvider: "openai",
  });
  const [modalField, setModalField] = useState<
    "bio" | "lore" | "style" | "messageExamples" | null
  >(null);
  const [modalInput, setModalInput] = useState<string>("");
  const [modalValues, setModalValues] = useState<string[]>([]);
  const [messageExamplesInput, setMessageExamplesInput] = useState<string>("");

  const openModal = (
    field: "bio" | "lore" | "style" | "messageExamples"
  ) => {
    setModalField(field);
    if (field === "style") {
      setModalValues(currentAgent.style?.all || []);
    } else {
      const arr = currentAgent[field] as string[] | undefined;
      setModalValues(arr || []);
    }
    if (field === "messageExamples") {
      setMessageExamplesInput(
        JSON.stringify(currentAgent.messageExamples, null, 2)
      );
    }
  };

  const closeModal = () => {
    setModalField(null);
    setModalValues([]);
    setModalInput("");
  };

  const saveModalValues = () => {
    if (modalField) {
      setCurrentAgent({
        ...currentAgent,
        [modalField === "style" ? "style" : modalField]:
          modalField === "style"
            ? { all: modalValues, chat: [], post: [] }
            : modalValues,
        messageExamples:
          modalField === "messageExamples"
            ? JSON.parse(messageExamplesInput || "[]")
            : currentAgent.messageExamples,
      });
    }
    closeModal();
  };

  const addAgent = () => {
    if (agents.length < 10 && currentAgent.name) {
      setAgents([
        ...agents,
        { id: agents.length + 1, ...currentAgent } as Agent,
      ]);
      setCurrentAgent({
        name: "",
        bio: [],
        lore: [],
        style: { all: [], chat: [], post: [] },
        messageExamples: [],
        plugins: [],
        clients: [],
        adjectives: [],
        topics: [],
        postExamples: [],
        modelProvider: "openai",
      });
    }
  };

  const removeAgent = (id: number) => {
    setAgents(agents.filter((agent) => agent.id !== id));
  };

  const launchChaosChain = async () => {
    setLoading(true);
    try {
      const launchPromises = agents.map(async (agent) => {
        try {
          const { id, ...payload } = agent;
          const response = await fetch(`${ELIZA_SERVER}/agent/start`, {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ characterJson: payload }),
          });

          if (response.ok) {
            console.log(`${agent.name} launched successfully!`);
          } else {
            console.error(`Failed to launch agent: ${agent.name}`);
          }
        } catch (error) {
          console.error(`Error starting the agent ${agent.name}:`, error);
        }
      });

      await Promise.all(launchPromises);
      alert("All agents launched successfully!");
    } catch (error) {
      console.error("Error launching agents:", error);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen bg-gradient-to-r from-blue-50 to-purple-50 p-8">
      {/* Header Section */}
      <header className="max-w-4xl mx-auto text-center mb-10">
        <h1 className="text-5xl font-extrabold text-gray-800">
          ChaosChain Agent Configurator
        </h1>
        <p className="text-lg text-gray-600 mt-4">
          Create and manage your agents with ease. Fill in the details and launch them into action!
        </p>
      </header>

      {/* Main Content Panels */}
      <div className="max-w-7xl mx-auto grid grid-cols-1 md:grid-cols-2 gap-8">
        {/* Form Panel */}
        <div className="bg-white p-6 rounded-lg shadow-lg">
          <h2 className="text-2xl font-bold text-gray-700 mb-4">
            Create New Agent
          </h2>
          <input
            type="text"
            placeholder="Agent Name"
            value={currentAgent.name || ""}
            onChange={(e) =>
              setCurrentAgent({ ...currentAgent, name: e.target.value })
            }
            className="w-full p-3 border rounded-md mb-4 focus:outline-none focus:ring-2 focus:ring-indigo-500"
          />

          {/* Modals for additional agent details */}
          <div className="grid grid-cols-2 gap-4 mb-4">
            {["bio", "lore", "style", "messageExamples"].map((field) => (
              <button
                key={field}
                onClick={() =>
                  openModal(
                    field as "bio" | "lore" | "style" | "messageExamples"
                  )
                }
                className="py-2 px-3 bg-gray-100 border border-gray-300 rounded-md text-gray-700 shadow-sm hover:bg-indigo-100 transition"
              >
                {field.charAt(0).toUpperCase() + field.slice(1)}
                <span className="ml-1 text-sm font-medium">
                  ({(currentAgent[field as keyof Agent] as string[])?.length || 0})
                </span>
              </button>
            ))}
          </div>

          <button
            onClick={addAgent}
            className="w-full py-3 bg-indigo-600 text-white rounded-md hover:bg-indigo-700 transition"
          >
            Add Agent
          </button>
        </div>

        {/* Agents List Panel */}
        <div className="bg-white p-6 rounded-lg shadow-lg">
          <h2 className="text-2xl font-bold text-gray-700 mb-4">
            Agents List
          </h2>
          {agents.length === 0 ? (
            <p className="text-gray-500">
              No agents added yet. Please add an agent.
            </p>
          ) : (
            <div className="space-y-4 max-h-96 overflow-y-auto pr-2">
              {agents.map((agent) => (
                <div
                  key={agent.id}
                  className="relative p-4 bg-gray-50 rounded-md shadow"
                >
                  <h3 className="text-xl font-semibold text-gray-800 mb-2">
                    {agent.name}
                  </h3>
                  <p className="text-sm text-gray-600 mb-1">
                    <strong>Bio:</strong> {agent.bio.join(", ")}
                  </p>
                  <p className="text-sm text-gray-600 mb-1">
                    <strong>Lore:</strong> {agent.lore.join(", ")}
                  </p>
                  <p className="text-sm text-gray-600">
                    <strong>Style:</strong>{" "}
                    {(agent.style as { all: string[] }).all.join(", ")}
                  </p>
                  <button
                    onClick={() => removeAgent(agent.id)}
                    className="absolute top-2 right-2 text-sm bg-red-500 text-white rounded-full px-2 py-1 hover:bg-red-600 transition"
                  >
                    ✖
                  </button>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>

      {/* Launch Button */}
      <div className="max-w-4xl mx-auto mt-10">
        <button
          onClick={launchChaosChain}
          className="w-full py-4 bg-green-600 text-white font-bold rounded-md hover:bg-green-700 shadow-lg transition"
        >
          {loading ? "Launching..." : "Launch ChaosChain"}
        </button>
      </div>

      {/* Modal for editing additional fields */}
      {modalField && (
        <div className="fixed inset-0 flex items-center justify-center bg-black bg-opacity-50 z-50">
          <div className="bg-white p-8 rounded-xl shadow-2xl w-full max-w-md">
            <h3 className="text-2xl font-semibold mb-6">
              Edit {modalField}
            </h3>

            {modalField === "messageExamples" ? (
              <textarea
                placeholder="Enter JSON Array"
                value={messageExamplesInput}
                onChange={(e) => setMessageExamplesInput(e.target.value)}
                className="w-full p-3 border border-gray-300 rounded-md h-28 focus:outline-none focus:ring-2 focus:ring-indigo-500 transition"
              ></textarea>
            ) : (
              <>
                <div className="flex space-x-3 mb-4">
                  <input
                    type="text"
                    placeholder={`Add ${modalField}`}
                    value={modalInput}
                    onChange={(e) => setModalInput(e.target.value)}
                    className="w-full p-3 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-indigo-500"
                  />
                  <button
                    onClick={() => {
                      if (modalInput.trim()) {
                        setModalValues([...modalValues, modalInput.trim()]);
                        setModalInput("");
                      }
                    }}
                    className="py-2 px-4 bg-blue-600 text-white rounded-md hover:bg-blue-700 transition"
                  >
                    Add
                  </button>
                </div>

                <div className="space-y-2">
                  {modalValues.map((value, index) => (
                    <div
                      key={index}
                      className="flex items-center justify-between bg-gray-100 p-2 rounded-md"
                    >
                      <span className="text-gray-700">{value}</span>
                      <button
                        onClick={() =>
                          setModalValues(modalValues.filter((_, i) => i !== index))
                        }
                        className="text-red-500 hover:text-red-600"
                      >
                        ✖
                      </button>
                    </div>
                  ))}
                </div>
              </>
            )}

            <div className="flex justify-end space-x-3 mt-6">
              <button
                onClick={closeModal}
                className="py-2 px-4 bg-gray-200 rounded-md hover:bg-gray-300 transition"
              >
                Cancel
              </button>
              <button
                onClick={saveModalValues}
                className="py-2 px-4 bg-indigo-600 text-white rounded-md hover:bg-indigo-700 transition"
              >
                Save
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
