export type MessageExample = {
  user: string;
  context: { text: string; action?: string };
};

export type Agent = {
  id: number;
  name: string;
  bio: string[];
  lore: string[];
  style: { all: string[]; chat: string[]; post: string[] };
  messageExamples: MessageExample[];
  plugins: string[];
  clients: string[];
  adjectives: string[];
  topics: string[];
  postExamples: string[];
  modelProvider: string;
};
