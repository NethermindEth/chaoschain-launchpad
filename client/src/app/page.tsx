import Head from "next/head";
import AgentConfigurator from "@/app/components/AgentConfigurator";

export default function Home() {
  return (
    <div className="w-full h-screen flex flex-col items-center justify-center bg-gray-100 pt-16">
      <Head>
        <title>ChaosChain Launchpad</title>
        <link href="https://fonts.googleapis.com/css2?family=Lato:ital,wght@0,100;0,300;0,400;0,700;0,900;1,100;1,300;1,400;1,700;1,900&display=swap" rel="stylesheet"/>
      </Head>
      <h1 className="text-4xl font-bold text-gray-900 m-6">
        ChaosChain Launchpad
      </h1>
      <AgentConfigurator />
    </div>
  );
}
