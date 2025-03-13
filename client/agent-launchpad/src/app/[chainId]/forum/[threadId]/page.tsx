"use client";

import { useParams, useSearchParams } from "next/navigation";
import ThreadDetailPage from "@/app/forum/[threadId]/page";

export default function ChainThreadDetailPage() {
  const params = useParams();
  const chainId = params.chainId as string;

  return <ThreadDetailPage chainId={chainId} />;
} 