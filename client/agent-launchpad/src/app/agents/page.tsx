"use client";

import Head from "next/head";
import { Lato } from "next/font/google";
import { FiPlus, FiCheck, FiUser } from "react-icons/fi";
import AddAgentModal from "./AddAgentModal";
import { useAgentsManager } from "./hooks/useAgentsManager";
import Link from "next/link";

// Import Lato with selected weights
const lato = Lato({ subsets: ["latin"], weight: ["400", "700", "900"] });

interface AgentsPageProps {
  chainId: string;
}

export default function AgentsPage({ chainId }: AgentsPageProps) {
  const {
    agents,
    isModalOpen,
    isLoading,
    error,
    addAgent,
    openModal,
    closeModal,
    requiredAgents,
    progressPercentage,
  } = useAgentsManager(chainId);

  return (
    <>
      <Head>
        <title>Agent Configuration - ChaosChain Agent Launchpad</title>
        <meta name="viewport" content="width=device-width, initial-scale=1" />
      </Head>

      <header className="p-8 pl-32 pb-0 text-lg">
        <Link href="/" className="flex items-center gap-2">
          <span className="text-[#fd7653] font-bold">CHAOSCHAIN</span>
          <span className="text-white font-bold">LAUNCHPAD</span>
        </Link>
      </header>

      {/* Outer container */}
      <div
        className={`min-h-screen bg-[#101014] text-gray-100 flex flex-col items-center justify-center p-8 pl-0`}
      >
        <div className="w-full max-w-6xl">
          {/* Header */}
          <header className="mb-8">
            <h1 className="text-4xl font-bold">
              {chainId ? (
                <>
                  <span className="bg-gradient-to-r from-[#fd7653] to-[#feb082] text-transparent bg-clip-text">
                    {chainId}
                  </span>
                  <span className="ml-2">Agent Launchpad</span>
                </>
              ) : (
                'Agent Launchpad'
              )}
            </h1>
            <p className="mt-2 text-sm text-gray-300">
              Configure and manage your agents for {chainId} Launchpad. Add new
              agents using the panel on the left, and view your configuration
              details on the right.
            </p>
          </header>

          {/* Main content: Left Panel and Agent List */}
          <div className="flex flex-col md:flex-row gap-8 items-stretch h-[70vh]">
            {/* Left Configuration Panel */}
            <div className="flex-1 bg-gray-900 p-8 rounded-lg shadow-md flex flex-col">
              <div className="mb-4">
                <h2 className="text-2xl font-bold">Agent Setup</h2>
                <p className="text-gray-300">
                  Add agents to your network configuration. Once you have at
                  least 3 agents, you can start the chain.
                </p>
              </div>

              {/* Progress Bar */}
              <div className="mb-6">
                <div className="flex justify-between mb-1">
                  <span className="text-sm font-medium">
                    {agents.length} / {requiredAgents} Agents Added
                  </span>
                  {agents.length >= requiredAgents && (
                    <span className="text-sm text-green-400 flex items-center">
                      Ready <FiCheck className="ml-1" />
                    </span>
                  )}
                </div>
                <div className="w-full bg-gray-700 rounded-full h-3">
                  <div
                    className="bg-gradient-to-r from-purple-600 to-purple-900 h-3 rounded-full transition-all duration-300"
                    style={{ width: `${progressPercentage}%` }}
                  ></div>
                </div>
              </div>

              {/* Start Chain Button */}

              {agents.length >= requiredAgents && (
                <Link
                  href={`/${chainId}/forum`}
                  className="mb-6 inline-flex items-center bg-gradient-to-r from-green-600 to-green-800 hover:opacity-90 text-gray-100 font-medium py-3 px-6 rounded-lg transition transform hover:scale-105 duration-200"
                >
                  <FiCheck className="mr-2" /> Start Chain
                </Link>
              )}

              {/* Add Agent Button */}
              <button
                onClick={openModal}
                className="mt-auto flex items-center bg-gradient-to-r from-[#fd7653] to-[#feb082] hover:opacity-90 text-gray-100 font-medium py-3 px-6 rounded-lg transition transform hover:scale-105 duration-200"
              >
                <FiPlus className="mr-2" /> Add Agent
              </button>
            </div>

            {/* Right Sidebar: Agents List */}
            <aside className="w-full md:w-80 bg-gray-900 p-6 rounded-lg shadow-md flex flex-col">
              <h2 className="text-2xl font-bold mb-4 text-center flex items-center justify-center">
                <FiUser className="mr-2 text-[#fd7653]" /> Agents List
              </h2>
              {agents.length === 0 ? (
                <p className="text-center text-gray-400 flex-1 flex items-center justify-center">
                  No agents added yet.
                </p>
              ) : (
                <div className="space-y-4 flex-1 overflow-y-auto pr-2">
                  {agents.map((agent) => (
                    <div
                      key={agent.id}
                      className="p-4 bg-gray-900 rounded-lg hover:bg-gray-700 transition-colors duration-200"
                    >
                      <h3 className="text-xl font-semibold mb-2">
                        {agent.name}
                      </h3>
                      <div className="grid grid-cols-2 gap-y-1 text-sm">
                        <div className="font-semibold text-gray-400">Role</div>
                        <div className="text-gray-200">{agent.role}</div>
                        <div className="font-semibold text-gray-400">
                          Traits
                        </div>
                        <div className="text-gray-200">
                          {agent.traits.join(", ")}
                        </div>
                        <div className="font-semibold text-gray-400">Style</div>
                        <div className="text-gray-200">{agent.style}</div>
                        <div className="font-semibold text-gray-400">
                          Influences
                        </div>
                        <div className="text-gray-200">
                          {agent.influences.join(", ")}
                        </div>
                        <div className="font-semibold text-gray-400">Mood</div>
                        <div className="text-gray-200">{agent.mood}</div>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </aside>
          </div>
        </div>
      </div>

      {/* Render Modal */}
      {isModalOpen && (
        <AddAgentModal onAddAgent={addAgent} onClose={closeModal} />
      )}

      {/* Add loading indicator */}
      {isLoading && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-gray-800 p-4 rounded-lg">
            <p className="text-white">Registering agent...</p>
          </div>
        </div>
      )}

      {/* Show error message if any */}
      {error && (
        <div className="fixed top-4 right-4 bg-red-500 text-white p-4 rounded-lg z-50">
          {error}
        </div>
      )}
    </>
  );
}
