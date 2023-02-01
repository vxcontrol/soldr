/* eslint-disable @typescript-eslint/naming-convention */
import {
    ModelsAgent,
    ModelsAgentInfo,
    ModelsAgentOS,
    ModelsAgentUpgradeTask,
    PrivateAgent,
    PrivateAgentDetails,
    PrivateAgents
} from '@soldr/api';
import { OperationSystemsList } from '@soldr/shared';

export type ConnectionStatus = 'connected' | 'disconnected';
export type AuthStatus = 'authorized' | 'unauthorized' | 'blocked';
export type UpgradeTaskStatus = 'new' | 'running' | 'ready' | 'failed';

export interface AgentInfo extends Omit<ModelsAgentInfo, 'os'> {
    os: OperationSystemsList;
}

export interface AgentUpgradeTask extends Omit<ModelsAgentUpgradeTask, 'status'> {
    status: UpgradeTaskStatus;
}

export interface AgentDetails extends Omit<PrivateAgentDetails, 'upgrade_task'> {
    upgrade_task?: AgentUpgradeTask;
}

export interface Agent extends Omit<ModelsAgent, 'info' | 'status' | 'auth_status'> {
    info: AgentInfo;
    details: AgentDetails;
    status: ConnectionStatus;
    auth_status: AuthStatus;
    _origin: ModelsAgent;
}

export const manyAgentsToModels = (data: PrivateAgents) =>
    data.agents.map(
        (agent: ModelsAgent) =>
            ({
                ...agent,
                info: { ...agent.info, os: toOsList(agent.info.os) },
                version: agent.version.replace(/^v/, ''),
                details: data.details.find((item) => item.hash === agent.hash),
                _origin: agent
            } as Agent)
    );

export const oneAgentToModel = (data: PrivateAgent) =>
    ({
        ...data.agent,
        info: { ...data.agent.info, os: toOsList(data.agent.info.os) },
        version: data.agent.version.replace(/^v/, ''),
        details: data.details,
        _origin: data.agent
    } as Agent);

export const agentToDto = (agent: Agent) =>
    ({
        ...agent._origin,
        description: agent.description,
        info: {
            ...agent._origin.info,
            tags: agent.info.tags
        }
    } as ModelsAgent);

export function toOsList(os: ModelsAgentOS): OperationSystemsList {
    return { [os.type]: [os.arch] } as OperationSystemsList;
}

export const removeHashFromVersion = (version: string) => (version || '').replace(/^v/, '').split('-')[0];

export const canUpgradeAgent = (agent: Agent, latestBinaryVersion: string) =>
    agent?.auth_status === 'authorized' &&
    latestBinaryVersion &&
    removeHashFromVersion(agent?.version) !== removeHashFromVersion(latestBinaryVersion) &&
    !['new', 'running'].includes(agent.details?.upgrade_task?.status);

export const isUpgradeAgentInProgress = (agent: Agent) =>
    ['new', 'running'].includes(agent?.details?.upgrade_task?.status);
