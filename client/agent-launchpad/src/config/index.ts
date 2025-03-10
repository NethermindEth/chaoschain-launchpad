export const API_CONFIG = {
    BASE_URL: process.env.NEXT_PUBLIC_API_URL || 'http://127.0.0.1:3000',
    AGENT_SERVICE_URL: process.env.NEXT_PUBLIC_AGENT_SERVICE_URL || 'http://localhost:5000',
    ENDPOINTS: {
        REGISTER: '/api/register',
    },
    TIMEOUT: 10000, // 10 seconds
} as const;

export const HTTP_STATUS = {
    OK: 200,
    BAD_REQUEST: 400,
    UNAUTHORIZED: 401,
    NOT_FOUND: 404,
    INTERNAL_SERVER_ERROR: 500,
} as const; 