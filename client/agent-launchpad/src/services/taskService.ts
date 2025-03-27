import { API_BASE_URL } from '@/config/constants';

export interface TaskSubmission {
  content: string;
  rewardPool: number;
}

export interface WorkSubmission {
  taskId: string;
  content: string;
  submittedBy: string;
}

export interface RewardProposal {
  taskId: string;
  totalReward: number;
  contributors: string[];
}

export const taskService = {
  submitTask: async (chainId: string, task: TaskSubmission) => {
    const response = await fetch(`${API_BASE_URL}/chains/${chainId}/tasks`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(task),
    });
    if (!response.ok) {
      throw new Error('Failed to submit task');
    }
    return response.json();
  },

  submitWork: async (chainId: string, work: WorkSubmission) => {
    const response = await fetch(`${API_BASE_URL}/chains/${chainId}/work-review`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(work),
    });
    if (!response.ok) {
      throw new Error('Failed to submit work');
    }
    return response.json();
  },

  proposeReward: async (chainId: string, proposal: RewardProposal) => {
    const response = await fetch(`${API_BASE_URL}/chains/${chainId}/rewards`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(proposal),
    });
    if (!response.ok) {
      throw new Error('Failed to propose reward distribution');
    }
    return response.json();
  },
}; 