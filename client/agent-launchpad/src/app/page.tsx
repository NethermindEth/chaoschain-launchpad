"use client";

import { DotLottieReact } from '@lottiefiles/dotlottie-react';
import type { NextPage } from 'next';
import Head from 'next/head';
import Link from 'next/link';
import Script from 'next/script';

const Home: NextPage = () => {
  return (
    <>
      <Head>
        <title>ChaosChain Agent Launchpad</title>
        <meta name="viewport" content="width=device-width, initial-scale=1" />
      </Head>

      <div className="min-h-screen flex flex-col items-center justify-center bg-gray-900 text-gray-100">
        <div className="text-center max-w-2xl px-4">
          <div className="mb-8">
            <DotLottieReact
              src="https://lottie.host/2ef09040-85b6-451f-81f8-fb1fe89b0f2f/mb6mvUSTGA.lottie"
              loop
              autoplay
            />
          </div>
          <h1 className="text-4xl font-extrabold mb-4">CHAOSCHAIN AGENT LAUNCHPAD</h1>
          <p className="text-base mb-6">
            Explore the future of blockchain by effortlessly creating, configuring, and launching agents.
            Experience real-time consensus and decentralized interactions.
          </p>
          <Link
            href="/agents"
            className="inline-block bg-gradient-to-r from-purple-700 to-purple-900 hover:opacity-90 text-gray-100 font-medium py-3 px-6 rounded-lg transition-all duration-300"
          >
            LET'S GET STARTED
          </Link>
        </div>
      </div>
    </>
  );
};

export default Home;