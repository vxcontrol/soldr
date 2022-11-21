/* eslint-disable @typescript-eslint/naming-convention */
import { DependencyType } from './dependency-type';
import { ModelsModuleA, ModelsModuleAShort } from './modules';
import { ModelsPolicy } from './policies';

export interface ModelsGroup {
    created_date?: string;
    hash: string;
    id?: number;
    info: ModelsGroupInfo;
}

export interface ModelsGroupDependency {
    min_agent_version?: string;
    min_module_version?: string;
    module_name?: string;
    policy_id?: number;
    source_module_name: string;
    status?: boolean;
    type: DependencyType;
}

export interface ModelsGroupInfo {
    name: ModelsGroupItemLocale;
    system?: boolean;
    tags: string[];
}

export interface ModelsGroupItemLocale {
    [lang: string]: string;
}

export interface PrivateGroup {
    details?: PrivateGroupDetails;
    group?: ModelsGroup;
}

export interface PrivateGroupDetails {
    active_modules?: number;
    agents?: number;
    consistency?: boolean;
    dependencies?: ModelsGroupDependency[];
    events_per_last_day?: number;
    hash?: string;
    joined_modules?: string;
    modules?: ModelsModuleAShort[];
    policies?: ModelsPolicy[];
}

export interface PrivateGroupInfo {
    from?: number;
    name?: string;
    tags?: string[];
}

export interface PrivateGroupModuleDetails {
    name?: string;
    policy?: ModelsPolicy;
    today?: number;
    total?: number;
    update?: boolean;
}

export interface PrivateGroupModules {
    details?: PrivateGroupModuleDetails[];
    modules?: ModelsModuleA[];
    total?: number;
}

export interface PrivateGroupPolicyPatch {
    /** Action on group policy must be one of activate, deactivate */
    action: 'activate' | 'deactivate';
    policy: ModelsPolicy;
}

export interface PrivateGroups {
    details?: PrivateGroupDetails[];
    groups?: ModelsGroup[];
    total?: number;
}

export enum GroupsSQLMappers {
    PolicyId = 'policy_id',
    PolicyName = 'policy_name',
    ModuleName = 'module_name'
}
