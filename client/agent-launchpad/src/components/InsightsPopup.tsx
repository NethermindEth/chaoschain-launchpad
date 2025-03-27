import React, { useState, useEffect } from 'react';
import { BarChart2, XCircle, TrendingUp, Users, MessageSquare } from 'lucide-react';
import { fetchInsights } from '@/services/api';

interface InsightsPopupProps {
  chainId: string;
}

const InsightsPopup: React.FC<InsightsPopupProps> = ({ chainId }) => {
  const [isOpen, setIsOpen] = useState(false);
  const [insights, setInsights] = useState<any>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (isOpen && !insights) {
      loadInsights();
    }
  }, [isOpen, chainId]);

  const loadInsights = async () => {
    try {
      setLoading(true);
      setError(null);
      const data = await fetchInsights(chainId);
      setInsights(data);
    } catch (err) {
      setError('Failed to load insights');
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  const insightCount = insights ? 
    Object.keys(insights).filter(key => 
      Array.isArray(insights[key]) ? insights[key].length > 0 : !!insights[key]
    ).length : 0;

  return (
    <div className="relative">
      <button
        onClick={() => setIsOpen(!isOpen)}
        className={`flex items-center gap-2 rounded-full px-3 py-1 text-white ${
          isOpen ? 'bg-blue-600' : 'bg-blue-500 hover:bg-blue-600'
        }`}
      >
        <BarChart2 size={16} />
        <span>{insightCount > 0 ? `${insightCount} Insights` : 'Insights'}</span>
        {isOpen && <XCircle size={16} onClick={(e) => { e.stopPropagation(); setIsOpen(false); }} />}
      </button>

      {isOpen && (
        <div className="absolute right-0 top-full mt-2 w-96 rounded-md bg-gray-900 p-4 shadow-lg z-50 border border-gray-800">
          <h3 className="mb-3 font-medium text-white flex items-center gap-2">
            <BarChart2 size={16} />
            Chain Insights
          </h3>
          
          {loading && <p className="text-gray-400">Loading insights...</p>}
          
          {error && <p className="text-red-400">{error}</p>}
          
          {insights && !loading && (
            <div className="space-y-4">
              {/* Blockchain Insights */}
              <div className="border-b border-gray-800 pb-3">
                <h4 className="text-sm font-medium text-gray-300 flex items-center gap-2 mb-2">
                  <TrendingUp size={14} />
                  Blockchain Metrics
                </h4>
                <div className="grid grid-cols-2 gap-2">
                  <div className="bg-gray-800 p-2 rounded">
                    <div className="text-xs text-gray-400">Block Rate</div>
                    <div className="text-white font-medium">
                      {insights.blockchainInsights?.blockProductionRate.toFixed(1)} blocks/min
                    </div>
                  </div>
                  <div className="bg-gray-800 p-2 rounded">
                    <div className="text-xs text-gray-400">Consensus Time</div>
                    <div className="text-white font-medium">
                      {insights.blockchainInsights?.consensusTime.toFixed(1)}s
                    </div>
                  </div>
                  <div className="bg-gray-800 p-2 rounded">
                    <div className="text-xs text-gray-400">Network Health</div>
                    <div className="text-white font-medium">
                      {insights.blockchainInsights?.networkHealth}
                    </div>
                  </div>
                  <div className="bg-gray-800 p-2 rounded">
                    <div className="text-xs text-gray-400">Participation</div>
                    <div className="text-white font-medium">
                      {(insights.blockchainInsights?.validatorParticipation * 100).toFixed(0)}%
                    </div>
                  </div>
                </div>
              </div>
              
              {/* Validator Insights */}
              <div className="border-b border-gray-800 pb-3">
                <h4 className="text-sm font-medium text-gray-300 flex items-center gap-2 mb-2">
                  <Users size={14} />
                  Validator Dynamics
                </h4>
                <div className="space-y-2">
                  {insights.validatorInsights?.slice(0, 3).map((validator: any) => (
                    <div key={validator.validatorId} className="bg-gray-800 p-2 rounded flex justify-between">
                      <div>
                        <div className="text-white font-medium">{validator.name}</div>
                        <div className="text-xs text-gray-400">
                          Influence: {(validator.influenceScore * 100).toFixed(0)}%
                        </div>
                      </div>
                      <div className="text-xs text-gray-400">
                        {validator.communicationStyle}
                      </div>
                    </div>
                  ))}
                </div>
              </div>
              
              {/* Forum Insights */}
              <div>
                <h4 className="text-sm font-medium text-gray-300 flex items-center gap-2 mb-2">
                  <MessageSquare size={14} />
                  Discussion Trends
                </h4>
                <div className="space-y-2">
                  {insights.forumInsights?.slice(0, 2).map((forum: any) => (
                    <div key={forum.discussionId} className="bg-gray-800 p-2 rounded">
                      <div className="flex justify-between">
                        <div className="text-white font-medium">Block {forum.blockId.substring(0, 8)}</div>
                        <div className="text-xs text-gray-400">
                          {forum.messageCount} messages
                        </div>
                      </div>
                      <div className="text-xs text-gray-400 mt-1">
                        Agreement: {(forum.agreementLevel * 100).toFixed(0)}% • Sentiment: {forum.sentiment}
                      </div>
                      {forum.topTopics.length > 0 && (
                        <div className="flex gap-1 mt-1">
                          {forum.topTopics.map((topic: string) => (
                            <span key={topic} className="text-xs bg-gray-700 px-1.5 py-0.5 rounded-full">
                              {topic}
                            </span>
                          ))}
                        </div>
                      )}
                    </div>
                  ))}
                </div>
              </div>
            </div>
          )}
          
          <button 
            onClick={loadInsights}
            className="mt-3 text-xs text-blue-400 hover:text-blue-300"
          >
            Refresh Insights
          </button>
        </div>
      )}
    </div>
  );
};

export default InsightsPopup; 