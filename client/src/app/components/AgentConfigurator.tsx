"use client";

import { Agent } from "@/types/agent";
import { useState } from "react";

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

  const openModal = (field: "bio" | "lore" | "style" | "messageExamples") => {
    setModalField(field);
    setModalValues(
      field === "style"
        ? currentAgent.style?.all || []
        : (currentAgent[field] as string[]) || []
    );
    setMessageExamplesInput(
      field === "messageExamples"
        ? JSON.stringify(currentAgent.messageExamples, null, 2)
        : ""
    );
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
    setLoading(true); // Start loading before the API calls

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

      await Promise.all(launchPromises); // Wait for all API calls to finish

      alert("All the Eliza agents successfully registered!");
    } catch (error) {
      console.error("Error launching ChaosChain:", error);
    } finally {
      setLoading(false); // Stop loading once all requests are done
    }
  };

  return (
    <div className="w-full h-screen flex flex-col items-center justify-start bg-white p-10">
      <h2 className="text-3xl font-bold text-gray-900 mb-6">
        Configure Agents
      </h2>

      {/* Input Fields */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-6 gap-4">
        <input
          type="text"
          placeholder="Agent Name"
          value={currentAgent.name || ""}
          onChange={(e) =>
            setCurrentAgent({ ...currentAgent, name: e.target.value })
          }
          className="p-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-indigo-500"
        />

        {/* Bio, Lore, Style, MessageExamples as Button-based Modals */}
        {["bio", "lore", "style", "messageExamples"].map((field) => (
          <button
            key={field}
            onClick={() =>
              openModal(field as "bio" | "lore" | "style" | "messageExamples")
            }
            className="px-4 py-2 bg-gray-300 text-black rounded-md hover:bg-gray-400 transition"
          >
            {field.charAt(0).toUpperCase() + field.slice(1)} (
            {(currentAgent[field as keyof Agent] as string[])?.length || 0})
          </button>
        ))}

        {/* Add agent button */}
        <button
          onClick={addAgent}
          className="px-4 py-2 bg-indigo-600 text-white rounded-md hover:bg-indigo-700 transition"
        >
          Add Agent
        </button>
      </div>

      {/* Agents Display */}
      <div className="flex flex-wrap items-center mt-6 space-x-4">
        {agents.map((agent) => (
          <div
            key={agent.id}
            className="relative bg-gray-100 p-4 rounded-md shadow-md w-60"
          >
            <h3 className="text-lg font-semibold">{agent.name}</h3>
            <p className="text-sm text-gray-700">
              <strong>Bio:</strong> {agent.bio.join(", ")}
            </p>
            <p className="text-sm text-gray-700">
              <strong>Lore:</strong> {agent.lore.join(", ")}
            </p>
            <p className="text-sm text-gray-700">
              <strong>Style:</strong> {agent.style.all.join(", ")}
            </p>
            <button
              onClick={() => removeAgent(agent.id)}
              className="absolute -top-2 -right-2 text-xs bg-red-500 text-white px-2 py-1 rounded-full"
            >
              ✖
            </button>
          </div>
        ))}
      </div>

      {/* Launch Button */}
      <div className="mt-6">
        <button
          onClick={launchChaosChain}
          className="w-full p-3 bg-green-600 text-white font-bold rounded-md hover:bg-green-700 transition"
        >
          {loading ? "Launching..." : "Launch ChaosChain"}
        </button>
      </div>

      {/* MODAL for Bio, Lore, Style, and Message Examples */}
      {modalField && (
        <div className="fixed inset-0 flex items-center justify-center bg-black bg-opacity-50">
          <div className="bg-white p-6 rounded-lg shadow-lg w-1/2">
            <h3 className="text-xl font-semibold mb-4">Edit {modalField}</h3>

            {/* Special case for messageExamples (JSON Input) */}
            {modalField === "messageExamples" ? (
              <textarea
                placeholder="Enter JSON Array"
                value={messageExamplesInput}
                onChange={(e) => setMessageExamplesInput(e.target.value)}
                className="w-full p-2 border border-gray-300 rounded-md h-24"
              ></textarea>
            ) : (
              <>
                <div className="flex space-x-2">
                  <input
                    type="text"
                    placeholder={`Add ${modalField}`}
                    className="w-full p-2 border border-gray-300 rounded-md"
                    value={modalInput}
                    onChange={(e) => setModalInput(e.target.value)}
                  />
                  <button
                    onClick={() => {
                      if (modalInput.trim()) {
                        setModalValues([...modalValues, modalInput.trim()]);
                        setModalInput("");
                      }
                    }}
                    className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700"
                  >
                    Add
                  </button>
                </div>

                <div className="mt-2">
                  {modalValues.map((value, index) => (
                    <div
                      key={index}
                      className="flex justify-between bg-gray-200 p-2 rounded-md mb-2"
                    >
                      {value}
                      <button
                        onClick={() =>
                          setModalValues(
                            modalValues.filter((_, i) => i !== index)
                          )
                        }
                        className="text-red-500"
                      >
                        ✖
                      </button>
                    </div>
                  ))}
                </div>
              </>
            )}

            {/* Modal Buttons */}
            <div className="flex justify-end space-x-2 mt-4">
              <button
                onClick={closeModal}
                className="px-4 py-2 bg-gray-300 rounded-md"
              >
                Cancel
              </button>
              <button
                onClick={saveModalValues}
                className="px-4 py-2 bg-indigo-600 text-white rounded-md"
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
