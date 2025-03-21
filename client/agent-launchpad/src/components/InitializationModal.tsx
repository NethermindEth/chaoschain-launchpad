import { useEffect, useState, useRef } from 'react';
import { wsService } from '@/services/websocket';
import { useRouter } from 'next/navigation';

interface InitializationModalProps {
  isOpen: boolean;
  onClose: () => void;
  chainId: string;
  totalAgents: number;
}

interface ChainCreatedEvent {
  chainId: string;
  timestamp: string;
}

interface AgentRegisteredEvent {
  agent: {
    name: string;
    id: string;
  };
  chainId: string;
  nodePort: number;
  timestamp: string;
}

enum InitState {
  CREATING_AGENTS,
  REGISTERING_AGENTS
}

export function InitializationModal({ isOpen, onClose, chainId, totalAgents }: InitializationModalProps) {
  const [events, setEvents] = useState<string[]>([]);
  const [registeredCount, setRegisteredCount] = useState(0);
  const [initState, setInitState] = useState<InitState>(InitState.CREATING_AGENTS);
  const router = useRouter();
  const eventsEndRef = useRef<HTMLDivElement>(null);

  const scrollToBottom = () => {
    eventsEndRef.current?.scrollIntoView({ behavior: "smooth" });
  };

  useEffect(() => {
    scrollToBottom();
  }, [events]); // Scroll when new events are added

  useEffect(() => {
    if (!isOpen) return;

    // Connect websocket when modal opens
    wsService.connect();

    const handleChainCreated = (data: ChainCreatedEvent) => {
      setEvents(prev => [...prev, `Chain created: ${data.chainId}`]);
    };

    const handleAgentRegistered = (data: AgentRegisteredEvent) => {
      if (registeredCount === 0) {
        setInitState(InitState.REGISTERING_AGENTS);
      }
      
      setEvents(prev => [...prev, `Agent registered: ${data.agent.name} (${data.agent.id})`]);
      setRegisteredCount(count => {
        const newCount = count + 1;
        if (newCount === totalAgents) {
          setTimeout(() => {
            onClose();
            router.push(`/${chainId}/agents`);
          }, 1000);
        }
        return newCount;
      });
    };

    wsService.subscribe('CHAIN_CREATED', handleChainCreated);
    wsService.subscribe('AGENT_REGISTERED', handleAgentRegistered);

    return () => {
      wsService.unsubscribe('CHAIN_CREATED', handleChainCreated);
      wsService.unsubscribe('AGENT_REGISTERED', handleAgentRegistered);
    };
  }, [isOpen, chainId, totalAgents, onClose, router]);

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center">
      <div className="bg-gray-900 p-8 rounded-xl w-[600px] max-h-[600px] overflow-y-auto scroll-smooth">
        <h2 className="text-3xl font-bold mb-6 text-white">Initializing Chaoschain</h2>
        <div className="space-y-4">
          {events.filter(event => event.includes('Chain created')).map((event, i) => (
            <div 
              key={i} 
              className="p-4 rounded-lg bg-[#fd7653] bg-opacity-20 border border-[#fd7653] text-white"
            >
              <div className="text-lg font-medium">{event}</div>
              <div className="text-sm text-gray-300 mt-2">
                {new Date().toLocaleTimeString()}
              </div>
            </div>
          ))}

          {initState === InitState.CREATING_AGENTS ? (
            <div className="flex flex-col items-center justify-center space-y-4 mt-4">
              <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-[#fd7653]"></div>
              <div className="text-lg text-white text-center">
                Creating agents best suited for your needs...
              </div>
            </div>
          ) : (
            <>
              {events.filter(event => !event.includes('Chain created')).map((event, i) => (
                <div 
                  key={i} 
                  className="p-4 rounded-lg bg-gray-800 border border-gray-700 text-white"
                >
                  <div className="text-lg font-medium">{event}</div>
                  <div className="text-sm text-gray-300 mt-2">
                    {new Date().toLocaleTimeString()}
                  </div>
                </div>
              ))}
            </>
          )}
          <div ref={eventsEndRef} />
        </div>
        <div className="mt-6 text-center text-base text-white font-medium">
          {initState === InitState.CREATING_AGENTS 
            ? "Generating unique personalities and traits..."
            : "Registering agents and initializing chain..."}
        </div>
      </div>
    </div>
  );
} 