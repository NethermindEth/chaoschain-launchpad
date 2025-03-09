"use client";

import Head from "next/head";
import Link from "next/link";
import { useParams } from "next/navigation";
import { FiArrowLeft } from "react-icons/fi";

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
  const { threadId } = params; 

  // const threadData = { ...threadData };

  return (
    <>
      <Head>
        <title>{threadData.title} - Thread Detail</title>
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
              src={`https://robohash.org/${encodeURIComponent(
                threadData.owner
              )}?size=80x80`}
              alt={threadData.owner}
              className="w-16 h-16 rounded-full border-2 border-indigo-500"
            />
            <div>
              <h1 className="text-3xl font-extrabold tracking-wide">
                {threadData.title}
              </h1>
              <p className="text-lg text-gray-400">
                Created by: {threadData.owner}
              </p>
            </div>
          </div>
          <p className="mt-6 text-lg leading-relaxed">
            {threadData.content}
          </p>
        </div>

        {/* Replies Section */}
        <div className="mt-10">
          <h2 className="text-2xl font-bold mb-4">Replies</h2>
          {threadData.replies.map((reply) => (
            <div
              key={reply.id}
              className="bg-gray-800 p-6 rounded-lg shadow-lg mb-4 transition transform hover:-translate-y-1 duration-200"
            >
              <div className="flex items-center space-x-4">
                <img
                  src={`https://robohash.org/${encodeURIComponent(
                    reply.author
                  )}?size=50x50`}
                  alt={reply.author}
                  className="w-12 h-12 rounded-full border-2 border-indigo-500"
                />
                <div>
                  <p className="text-xl font-bold">{reply.author}</p>
                  <p className="text-sm text-gray-400">
                    {new Date(reply.postedAt).toLocaleString()}
                  </p>
                </div>
              </div>
              <p className="mt-4 text-gray-300">{reply.content}</p>
            </div>
          ))}
          {threadData.replies.length === 0 && (
            <p className="text-gray-400 text-lg">No replies yet.</p>
          )}
        </div>
      </div>
    </>
  );
}