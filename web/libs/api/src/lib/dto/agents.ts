/* eslint-disable @typescript-eslint/naming-convention */
import { DependencyType } from './dependency-type';
import { ModelsGroup } from './groups';
import { ModelsModuleA, ModelsModuleAShort } from './modules';
import { ModelsPolicy } from './policies';

export interface ModelsAgent {
    auth_status: string;
    connected_date?: string;
    created_date?: string;
    description: string;
    group_id?: number;
    hash: string;
    id?: number;
    info: ModelsAgentInfo;
    ip: string;
    status: string;
    version: string;
}

export interface ModelsAgentDependency {
    min_agent_version?: string;
    min_module_version?: string;
    module_name?: string;
    policy_id?: number;
    source_module_name: string;
    status?: boolean;
    type: DependencyType;
}

export interface ModelsAgentInfo {
    net: ModelsAgentNet;
    os: ModelsAgentOS;
    tags: string[];
    users: ModelsAgentUser[];
}

export interface ModelsAgentNet {
    hostname: string;
    ips: string[];
}

export interface ModelsAgentOS {
    arch: string;
    name: string;
    type: string;
}

export interface ModelsAgentUpgradeTask {
    agent_id?: number;
    batch: string;
    created?: string;
    id?: number;
    last_update?: string;
    reason?: string;
    status: string;
    version: string;
}

export interface ModelsAgentUser {
    groups: string[];
    name: string;
}

export interface PrivateAgent {
    agent?: ModelsAgent;
    details?: PrivateAgentDetails;
}

export interface PrivateAgentDetails {
    active_modules?: number;
    consistency?: boolean;
    dependencies?: ModelsAgentDependency[];
    events_per_last_day?: number;
    group?: ModelsGroup;
    hash?: string;
    joined_modules?: string;
    modules?: ModelsModuleAShort[];
    policies?: ModelsPolicy[];
    upgrade_task?: ModelsAgentUpgradeTask;
}

export interface PrivateAgentInfo {
    arch: '386' | 'amd64';
    name: string;
    os: 'windows' | 'linux' | 'darwin';
}

export interface PrivateAgentModuleDetails {
    name?: string;
    policy?: ModelsPolicy;
    today?: number;
    total?: number;
    update?: boolean;
}

export interface PrivateAgentModules {
    details?: PrivateAgentModuleDetails[];
    modules?: ModelsModuleA[];
    total?: number;
}

export interface PrivateAgents {
    agents?: ModelsAgent[];
    details?: PrivateAgentDetails[];
    total?: number;
}

export interface PrivateAgentsAction {
    action: 'authorize' | 'block' | 'delete' | 'unauthorize' | 'move';
    filters?: UtilsTableFilter[];
    to?: number;
}

export interface PrivateAgentsActionResult {
    total?: number;
}

export interface UtilsTableFilter {
    field: string;
    value: any;
}

export enum AgentAction {
    Authorize = 'authorize',
    Block = 'block',
    Delete = 'delete',
    Unauthorize = 'unauthorize',
    Move = 'move',
    Edit = 'edit'
}

export interface PrivatePatchAgentAction {
    action: AgentAction;
    agent: ModelsAgent;
}

export enum AgentsSQLMappers {
    GroupId = 'group_id',
    GroupName = 'group_name',
    ModuleName = 'module_name',
    Os = 'os',
    PolicyId = 'policy_id',
    Version = 'version'
}
