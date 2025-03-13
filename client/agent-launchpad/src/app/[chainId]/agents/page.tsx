"use client";

import { useParams } from 'next/navigation';
import AgentsPage from '@/app/agents/page';

export default function ChainAgentsPage() {
  const params = useParams();
  const chainId = params.chainId as string;

  // Pass chainId to the AgentsPage component
  return <AgentsPage />;
} 