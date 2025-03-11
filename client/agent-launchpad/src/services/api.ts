import { API_CONFIG } from '@/config';

interface RegisterAgentParams {
    name: string;
    role: "producer" | "validator";
    traits: string[];
    style: string;
    influences: string[];
    mood: string;
}

interface RegisterAgentResponse {
    agentID: string;
    p2pPort: number;
    apiPort: number;
    message: string;
}

export class ApiError extends Error {
    constructor(
        message: string,
        public status?: number,
        public data?: any
    ) {
        super(message);
        this.name = 'ApiError';
    }
}

export async function registerAgent(agent: RegisterAgentParams): Promise<RegisterAgentResponse> {
    try {
        const response = await fetch(`${API_CONFIG.BASE_URL}${API_CONFIG.ENDPOINTS.REGISTER}`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({
                ...agent,
                api_key: process.env.NEXT_PUBLIC_OPENAI_API_KEY,
                endpoint: `${API_CONFIG.AGENT_SERVICE_URL}/${agent.role}`
            }),
        });

        const data = await response.json();

        if (!response.ok) {
            throw new ApiError(
                data.error || 'Failed to register agent',
                response.status,
                data
            );
        }

        return data as RegisterAgentResponse;
    } catch (error) {
        if (error instanceof ApiError) {
            throw error;
        }
        throw new ApiError(
            error instanceof Error ? error.message : 'Unknown error occurred'
        );
    }
} 