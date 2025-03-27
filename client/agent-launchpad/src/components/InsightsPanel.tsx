import React, { useState, useEffect } from 'react';
import { fetchDiscussionAnalysis } from '@/services/api';

interface InsightsPanelProps {
    chainId: string;
}

const InsightsPanel: React.FC<InsightsPanelProps> = ({ chainId }) => {
    const [analysis, setAnalysis] = useState<string>('');
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);

    useEffect(() => {
        const loadAnalysis = async () => {
            try {
                setLoading(true);
                setError(null);
                const data = await fetchDiscussionAnalysis(chainId);
                const cleanedAnalysis = data.analysis.replace(/^# Discussion Analysis\n+/i, '');
                setAnalysis(cleanedAnalysis);
            } catch (err) {
                setError('Failed to load analysis');
                console.error(err);
            } finally {
                setLoading(false);
            }
        };

        loadAnalysis();
    }, [chainId]);

    return (
        <div>
            {loading && (
                <div className="bg-gray-800 p-6 rounded-lg">
                    Loading analysis...
                </div>
            )}
            
            {error && (
                <div className="bg-gray-800 p-6 rounded-lg text-red-400">
                    {error}
                </div>
            )}
            
            {analysis && !loading && (
                <div className="mt-8 bg-gray-800 p-6 rounded-lg">
                    <pre className="bg-gray-900 p-4 rounded overflow-auto whitespace-pre-wrap">
                        {analysis}
                    </pre>
                </div>
            )}
        </div>
    );
};

export default InsightsPanel; 