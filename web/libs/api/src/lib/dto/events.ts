/* eslint-disable @typescript-eslint/naming-convention */
import { ModelsAgent } from './agents';
import { ModelsGroup } from './groups';
import { ModelsModuleAShort } from './modules';
import { ModelsPolicy } from './policies';

export interface ModelsEvent {
    agent_id?: number;
    date?: string;
    id?: number;
    info: ModelsEventInfo;
    module_id?: number;
}

export interface ModelsEventInfo {
    actions?: string[];
    data: Record<string, any>;
    name: string;
    time?: number;
    uniq: string;
}

export interface PrivateEvents {
    agents?: ModelsAgent[];
    events?: ModelsEvent[];
    groups?: ModelsGroup[];
    modules?: ModelsModuleAShort[];
    policies?: ModelsPolicy[];
    total?: number;
}

export enum EventsSQLMappers {
    AgentId = 'agent_id',
    PolicyId = 'policy_id',
    ModuleId = 'module_id',
    GroupId = 'group_id'
}
