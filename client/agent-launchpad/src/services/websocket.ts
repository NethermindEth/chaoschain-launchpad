type WebSocketCallback = (event: any) => void;

class WebSocketService {
    private ws: WebSocket | null = null;
    private subscribers: { [eventType: string]: WebSocketCallback[] } = {};

    connect() {
        this.ws = new WebSocket('ws://localhost:3000/ws');
        
        this.ws.onmessage = (event) => {
            const data = JSON.parse(event.data);
            const callbacks = this.subscribers[data.type] || [];
            callbacks.forEach(callback => callback(data.payload));
        };
    }

    subscribe(eventType: string, callback: WebSocketCallback) {
        if (!this.subscribers[eventType]) {
            this.subscribers[eventType] = [];
        }
        this.subscribers[eventType].push(callback);
    }

    unsubscribe(eventType: string, callback: WebSocketCallback) {
        this.subscribers[eventType] = this.subscribers[eventType]?.filter(cb => cb !== callback) || [];
    }
}

export const wsService = new WebSocketService(); 