export const API_CONFIG = {
  BASE_URL: "http://localhost:3000/api",
  WS_URL: process.env.NEXT_PUBLIC_WS_URL || "ws://localhost:3000/ws",
  ENDPOINTS: {
    REGISTER_AGENT: "/register",
    CREATE_CHAIN: "/chains",
    FETCH_CHAINS: "/chains",
    FETCH_VALIDATORS: "/validators",
    PROPOSE_BLOCK: "/block/propose",
    SUBMIT_TRANSACTION: "/transactions",
    FETCH_INSIGHTS: "/insights"
  },
  AGENT_SERVICE_URL: "http://localhost:3000/api/agent"
}; 