import { useState, useEffect } from 'react';
import { FiX } from 'react-icons/fi';

interface Agent {
    ID: string;
    Name: string;
}

interface TransactionModalProps {
    onClose: () => void;
    onSubmit: (transaction: any) => void;
}

export default function TransactionModal({ onClose, onSubmit }: TransactionModalProps) {
    const [agents, setAgents] = useState<Agent[]>([]);
    const [formData, setFormData] = useState({
        from: '',
        to: '',
        amount: 0,
        fee: 0,
        content: '',
        timestamp: Math.floor(Date.now() / 1000) // Convert to epoch seconds
    });

    useEffect(() => {
        const fetchAgents = async () => {
            try {
                const response = await fetch('http://127.0.0.1:3000/api/validators');
                const data = await response.json();
                setAgents(data.validators);
            } catch (error) {
                console.error('Failed to fetch agents:', error);
            }
        };
        fetchAgents();
    }, []);

    const handleSubmit = (e: React.FormEvent) => {
        e.preventDefault();
        onSubmit(formData);
        onClose();
    };

    return (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
            <div className="bg-gray-800 p-6 rounded-lg w-full max-w-md">
                <div className="flex justify-between items-center mb-4">
                    <h2 className="text-xl font-bold">Propose Transaction</h2>
                    <button onClick={onClose} className="text-gray-400 hover:text-white">
                        <FiX size={24} />
                    </button>
                </div>
                
                <form onSubmit={handleSubmit} className="space-y-4">
                    <div>
                        <label className="block text-sm font-medium mb-1">From</label>
                        <select
                            value={formData.from}
                            onChange={(e) => setFormData({...formData, from: e.target.value})}
                            className="w-full bg-gray-700 rounded p-2"
                            required
                        >
                            <option value="">Select agent</option>
                            {agents.map(agent => (
                                <option key={agent.ID} value={agent.ID}>{agent.Name}</option>
                            ))}
                        </select>
                    </div>

                    <div>
                        <label className="block text-sm font-medium mb-1">To</label>
                        <select
                            value={formData.to}
                            onChange={(e) => setFormData({...formData, to: e.target.value})}
                            className="w-full bg-gray-700 rounded p-2"
                            required
                        >
                            <option value="">Select agent</option>
                            {agents.map(agent => (
                                <option key={agent.ID} value={agent.ID}>{agent.Name}</option>
                            ))}
                        </select>
                    </div>

                    <div>
                        <label className="block text-sm font-medium mb-1">Amount</label>
                        <input
                            type="number"
                            value={formData.amount}
                            onChange={(e) => setFormData({...formData, amount: parseInt(e.target.value)})}
                            className="w-full bg-gray-700 rounded p-2"
                            required
                        />
                    </div>

                    <div>
                        <label className="block text-sm font-medium mb-1">Fee</label>
                        <input
                            type="number"
                            value={formData.fee}
                            onChange={(e) => setFormData({...formData, fee: parseInt(e.target.value)})}
                            className="w-full bg-gray-700 rounded p-2"
                            required
                        />
                    </div>

                    <div>
                        <label className="block text-sm font-medium mb-1">Timestamp</label>
                        <input
                            type="text"
                            value={formData.timestamp}
                            className="w-full bg-gray-700 rounded p-2"
                            disabled
                        />
                    </div>

                    <div>
                        <label className="block text-sm font-medium mb-1">Content</label>
                        <textarea
                            value={formData.content}
                            onChange={(e) => setFormData({...formData, content: e.target.value})}
                            className="w-full bg-gray-700 rounded p-2"
                            required
                            rows={3}
                        />
                    </div>

                    <button
                        type="submit"
                        className="w-full bg-purple-600 hover:bg-purple-700 text-white font-bold py-2 px-4 rounded"
                    >
                        Submit Transaction
                    </button>
                </form>
            </div>
        </div>
    );
} 