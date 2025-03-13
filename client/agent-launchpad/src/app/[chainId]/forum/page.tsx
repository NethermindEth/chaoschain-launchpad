"use client";

import { useParams } from 'next/navigation';
import ForumPage from '@/app/forum/page';

export default function ChainForumPage() {
  const params = useParams();
  const chainId = params.chainId as string;

  return <ForumPage chainId={chainId} />;
} 