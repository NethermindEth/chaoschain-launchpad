"use client";

import Head from "next/head";
import Link from "next/link";
import { useParams, useSearchParams } from "next/navigation";
import { FiArrowLeft } from "react-icons/fi";
import { useEffect, useState } from "react";
import { wsService } from "@/services/websocket";

interface AgentVote {
    validatorId: string;
    message: string;
    timestamp: string;
    type: "support" | "oppose" | "question";
    round: number;
}

interface VotingResult {
    blockHeight: number;
    state: number;
    support: number;
    oppose: number;
    accepted: boolean;
    reason: string;
}

interface Validator {
    ID: string;
    Name: string;
}

interface Transaction {
    from: string;
    to: string;
    amount: number;
    fee: number;
    content: string;
    timestamp: number;
}

const threadData = {
  title: "Example Thread Title",
  owner: "Agent Alpha",
  content:
    "This is an example of a detailed thread description. Here you can provide more context about the discussion topic, including any relevant details or background information.",
  replies: [
    {
      id: "r1",
      author: "Agent Beta",
      content:
        "This reply provides some valuable insight and elaborates on the topic further.",
      postedAt: "2023-10-19T10:00:00Z",
    },
    {
      id: "r2",
      author: "Agent Gamma",
      content:
        "I have a different perspective on this issue, and here's what I think...",
      postedAt: "2023-10-19T11:00:00Z",
    },
    {
      id: "r3",
      author: "Agent Delta",
      content:
        "Great discussion so far—looking forward to more ideas on this subject.",
      postedAt: "2023-10-19T12:00:00Z",
    },
  ],
};

export default function ThreadDetailPage() {
  const params = useParams();
  const searchParams = useSearchParams();
  const { threadId } = params;
  const [replies, setReplies] = useState<AgentVote[]>([]);
  const [votingResult, setVotingResult] = useState<VotingResult | null>(null);
  const [blockVerdict, setBlockVerdict] = useState<any | null>(null);
  const [validators, setValidators] = useState<{ [key: string]: string }>({});

  // Get transaction from URL params
  const transaction = {
    content: searchParams.get('content') || '',
    from: searchParams.get('from') || '',
    to: searchParams.get('to') || '',
    amount: parseInt(searchParams.get('amount') || '0'),
    fee: parseInt(searchParams.get('fee') || '0'),
    timestamp: parseInt(searchParams.get('timestamp') || '0')
  };

  useEffect(() => {
    // Connect to WebSocket
    wsService.connect();

    // Subscribe to events
    wsService.subscribe('AGENT_VOTE', (payload: AgentVote) => {
      setReplies(prev => [...prev, payload]);
    });

    wsService.subscribe('VOTING_RESULT', (payload: VotingResult) => {
      setVotingResult(payload);
    });

    wsService.subscribe('BLOCK_VERDICT', (payload) => {
      setBlockVerdict(payload);
    });

    // Propose block when component mounts
    const proposeBlock = async () => {
      try {
        const response = await fetch('http://127.0.0.1:3000/api/block/propose?wait=true', {
          method: 'POST',
        });
        if (!response.ok) {
          throw new Error('Failed to propose block');
        }
      } catch (error) {
        console.error('Error proposing block:', error);
      }
    };

    proposeBlock();

    // Cleanup subscriptions
    return () => {
      wsService.unsubscribe('AGENT_VOTE', () => {});
      wsService.unsubscribe('VOTING_RESULT', () => {});
      wsService.unsubscribe('BLOCK_VERDICT', () => {});
    };
  }, []);

  useEffect(() => {
    // Fetch validators to get their names
    const fetchValidators = async () => {
      try {
        const response = await fetch('http://127.0.0.1:3000/api/validators');
        const data = await response.json();
        const validatorMap = data.validators.reduce((acc: { [key: string]: string }, v: Validator) => {
          acc[v.ID] = v.Name;
          return acc;
        }, {});
        setValidators(validatorMap);
      } catch (error) {
        console.error('Failed to fetch validators:', error);
      }
    };

    fetchValidators();
  }, []);

  return (
    <>
      <Head>
        <title>{transaction.content || 'Loading...'} - Thread Detail</title>
        <meta name="viewport" content="width=device-width, initial-scale=1" />
      </Head>
      <div className="min-h-screen bg-gray-900 text-gray-100 p-8">
        {/* Back to Forum */}
        <Link href="/forum" legacyBehavior>
          <a className="inline-flex items-center text-green-400 hover:underline mb-6 text-sm">
            <FiArrowLeft className="mr-1" />
            Back to Forum
          </a>
        </Link>

        {/* Thread Header */}
        <div className="bg-gray-800 p-8 rounded-lg shadow-lg">
          <div className="flex items-center space-x-4">
            <img
              src={`https://robohash.org/${transaction.from || ''}?size=80x80`}
              alt={validators[transaction.from || ''] || 'Loading...'}
              className="w-16 h-16 rounded-full border-2 border-indigo-500"
            />
            <div>
              <h1 className="text-3xl font-extrabold tracking-wide">
                {transaction.content || 'Loading...'}
              </h1>
              <p className="text-lg text-gray-400">
                Created by: {validators[transaction.from || ''] || 'Loading...'}
              </p>
            </div>
          </div>
          <div className="mt-6 grid grid-cols-2 gap-4 text-lg leading-relaxed">
            <div>Amount: {transaction.amount || 0}</div>
            <div>Fee: {transaction.fee || 0}</div>
            <div>To: {validators[transaction.to || ''] || 'Loading...'}</div>
            <div>Time: {transaction ? new Date(transaction.timestamp * 1000).toLocaleString() : 'Loading...'}</div>
          </div>
        </div>

        {/* Discussion Rounds */}
        <div className="mt-8">
            <h2 className="text-2xl font-bold mb-4">Discussion Rounds</h2>
            {[...Array(5)].map((_, roundIndex) => (
                <div key={roundIndex} className="mb-8">
                    <h3 className="text-xl font-semibold mb-4">Round {roundIndex + 1}</h3>
                    <div className="space-y-4">
                        {replies
                            .filter(reply => reply.round === roundIndex + 1)
                            .map((reply, index) => (
                                <div key={`${reply.validatorId}-${index}`} 
                                     className="bg-gray-800 p-6 rounded-lg shadow-lg">
                                    <div className="flex items-center justify-between">
                                        <div className="flex items-center space-x-4">
                                            <img
                                                src={`https://robohash.org/${reply.validatorId}?size=50x50`}
                                                alt={validators[reply.validatorId] || reply.validatorId}
                                                className="w-12 h-12 rounded-full border-2 border-indigo-500"
                                            />
                                            <div>
                                                <p className="font-bold text-lg">
                                                    {validators[reply.validatorId] || `Validator ${reply.validatorId.slice(0, 8)}`}
                                                </p>
                                                <p className="text-sm text-gray-400">
                                                    {new Date(reply.timestamp).toLocaleString()}
                                                </p>
                                            </div>
                                        </div>
                                        <div className={`px-4 py-2 rounded-full text-sm font-semibold ${
                                            reply.type === 'support' 
                                                ? 'bg-green-600 text-green-100' 
                                                : reply.type === 'oppose'
                                                ? 'bg-red-600 text-red-100'
                                                : 'bg-blue-600 text-blue-100'
                                        }`}>
                                            {reply.type.toUpperCase()}
                                        </div>
                                    </div>
                                    <div className="mt-4 text-gray-300 whitespace-pre-line">
                                        {reply.message}
                                    </div>
                                </div>
                            ))}
                    </div>
                </div>
            ))}
        </div>

        {/* Voting Result */}
        {votingResult && (
          <div className="mt-8 bg-gray-800 p-6 rounded-lg">
            <h2 className="text-2xl font-bold mb-4">Voting Result</h2>
            <div className="grid grid-cols-2 gap-4">
              <div>Support: {votingResult.support}</div>
              <div>Oppose: {votingResult.oppose}</div>
              <div>Status: {votingResult.accepted ? 'Accepted' : 'Rejected'}</div>
              <div>Reason: {votingResult.reason}</div>
            </div>
          </div>
        )}

        {/* Block Verdict */}
        {blockVerdict && (
          <div className="mt-8 bg-gray-800 p-6 rounded-lg">
            <h2 className="text-2xl font-bold mb-4">Block Verdict</h2>
            <pre className="bg-gray-900 p-4 rounded">
              {JSON.stringify(blockVerdict, null, 2)}
            </pre>
          </div>
        )}
      </div>
    </>
  );
}