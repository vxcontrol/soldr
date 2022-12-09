/* eslint-disable @typescript-eslint/naming-convention */
import { DependencyType } from './dependency-type';
import { ModelsGroup } from './groups';
import { ModelsModuleA, ModelsModuleAShort } from './modules';

export interface ModelsPolicy {
    created_date?: string;
    hash: string;
    id?: number;
    info: ModelsPolicyInfo;
}

export interface ModelsPolicyDependency {
    min_agent_version?: string;
    min_module_version?: string;
    module_name?: string;
    source_module_name: string;
    status?: boolean;
    type: DependencyType;
}

export interface ModelsPolicyInfo {
    name: ModelsPolicyItemLocale;
    system?: boolean;
    tags: string[];
}

export interface ModelsPolicyItemLocale {
    [lang: string]: string;
}

export interface PrivatePolicies {
    details?: PrivatePolicyDetails[];
    policies?: ModelsPolicy[];
    total?: number;
}

export interface PrivatePolicy {
    details?: PrivatePolicyDetails;
    policy?: ModelsPolicy;
}

export interface PrivatePolicyDetails {
    active_modules?: number;
    agents?: number;
    consistency?: boolean;
    dependencies?: ModelsPolicyDependency[];
    events_per_last_day?: number;
    groups?: ModelsGroup[];
    hash?: string;
    joined_modules?: string;
    modules?: ModelsModuleAShort[];
    update_modules?: boolean;
}

export interface PrivatePolicyGroupPatch {
    /** Action on policy group must be one of activate, deactivate */
    action: 'activate' | 'deactivate';
    group: ModelsGroup;
}

export interface PrivatePolicyInfo {
    from?: number;
    name?: string;
    tags?: string[];
}

export interface PrivatePolicyModuleDetails {
    active?: boolean;
    exists?: boolean;
    name?: string;
    today?: number;
    total?: number;
    update?: boolean;
    duplicate?: boolean;
}

export interface PrivatePolicyModulePatch {
    /** Action on group module must be one of activate, deactivate, update, store */
    action: 'activate' | 'deactivate' | 'store' | 'update';
    module?: ModelsModuleA;
    version?: string;
}

export interface PrivatePolicyModules {
    details?: PrivatePolicyModuleDetails[];
    modules?: ModelsModuleA[];
    total?: number;
}

export interface PrivatePolicyModulesUpdates {
    modules?: ModelsModuleA[];
    policies?: ModelsPolicy[];
}

export interface PrivatePolicyCountResponse {
    all: number;
    without_groups: number;
}

export enum PoliciesSQLMappers {
    ModuleName = 'module_name',
    GroupId = 'group_id'
}
