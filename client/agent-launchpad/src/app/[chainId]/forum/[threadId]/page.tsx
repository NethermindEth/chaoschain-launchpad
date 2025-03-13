"use client";

import { useParams } from "next/navigation";
import ThreadDetailPage from "@/app/forum/[threadId]/page";

export default function ChainThreadDetailPage() {
  const params = useParams();
  const chainId = params.chainId as string;
  const threadId = params.threadId as string;

  // Pass both parameters to the ThreadDetailPage component
  return <ThreadDetailPage />;
} 