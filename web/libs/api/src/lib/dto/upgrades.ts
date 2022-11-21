import { ModelsAgent, ModelsAgentUpgradeTask, UtilsTableFilter } from './agents';
import { ModelsGroup } from './groups';

export interface PrivateUpgradeAgent {
    details?: PrivateUpgradeAgentDetails;
    task?: ModelsAgentUpgradeTask;
}

export interface PrivateUpgradeAgentDetails {
    agent?: ModelsAgent;
    group?: ModelsGroup;
}

export interface PrivateUpgradesAgents {
    tasks?: ModelsAgentUpgradeTask[];
    total?: number;
}

export interface PrivateUpgradesAgentsAction {
    filters?: UtilsTableFilter[];
    version: string;
}

export interface PrivateUpgradesAgentsActionResult {
    batch?: string;
    total?: number;
}
