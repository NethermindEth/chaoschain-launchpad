import Head from "next/head";
import AgentConfigurator from "@/app/components/AgentConfigurator";

export default function Home() {
  return (
    <div className="w-full h-screen flex flex-col items-center justify-center bg-gray-100">
      <Head>
        <title>ChaosChain Launchpad</title>
      </Head>
      <h1 className="text-4xl font-bold text-gray-900 m-6">
        ChaosChain Launchpad
      </h1>
      <AgentConfigurator />
    </div>
  );
}
