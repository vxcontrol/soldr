/* eslint-disable @typescript-eslint/naming-convention */
import { NcformSchema } from '@soldr/shared';

import { DependencyType } from './dependency-type';

export type ModelsChangelog = Record<string, ModelsChangelogVersion>;

export interface ModelsChangelogDesc {
    date: string;
    description: string;
    title: string;
}

export type ModelsChangelogVersion = Record<string, ModelsChangelogDesc>;

export interface ModelsDependencyItem {
    min_agent_version?: string;
    min_module_version?: string;
    module_name?: string;
    type: DependencyType;
}

export type ModelsEventConfig = Record<string, ModelsEventConfigItem>;

export type ModelsActionConfig = Record<string, ModelsActionConfigItem>;

export type ModelsSecureModuleConfig = Record<string, ModelsModuleSecureParameter>;

export interface ModelsActionConfigItem {
    fields?: string[];
    priority: number;
}

export interface ModelsEventConfigAction {
    fields?: string[];
    module_name: string;
    name: string;
    priority: number;
}

export interface ModelsEventConfigItem {
    actions: ModelsEventConfigAction[];
    fields?: string[];
    group_by?: string[];
    max_count?: number;
    max_time?: number;
    seq?: ModelsEventConfigSeq[];
    type: string;
    [key: string]: any;
}

export interface ModelsEventConfigSeq {
    min_count: number;
    name: string;
}

export interface ModelsLocale {
    action_config: Record<string, Record<string, ModelsModuleLocaleDesc>>;
    actions: Record<string, ModelsModuleLocaleDesc>;
    actions_additional_args: Record<string, Record<string, string>>;
    config: Record<string, ModelsModuleLocaleDesc>;
    config_additional_args: Record<string, Record<string, string>>;
    event_config: Record<string, Record<string, ModelsModuleLocaleDesc>>;
    events: Record<string, ModelsModuleLocaleDesc>;
    events_additional_args: Record<string, Record<string, string>>;
    fields: Record<string, ModelsModuleLocaleDesc>;
    fields_additional_args: Record<string, Record<string, string>>;
    module: ModelsModuleLocaleDesc;
    secure_config: Record<string, ModelsModuleLocaleDesc>;
    secure_config_additional_args: Record<string, Record<string, string>>;
    tags: Record<string, ModelsModuleLocaleDesc>;
    ui: Record<string, Record<string, string>>;
}

export interface ModelsLocaleDesc {
    description: string;
    title: string;
}

export interface ModelsModuleA {
    action_config_schema: any;
    changelog: ModelsChangelog;
    config_schema: NcformSchema;
    current_action_config: ModelsActionConfig;
    current_config: ModelsModuleConfig;
    current_event_config: ModelsEventConfig;
    default_action_config: ModelsActionConfig;
    default_config: ModelsModuleConfig;
    default_event_config: ModelsEventConfig;
    dynamic_dependencies: ModelsDependencyItem[];
    event_config_schema: NcformSchema;
    fields_schema: NcformSchema;
    id?: number;
    info: ModelsModuleInfo;
    join_date?: string;
    last_module_update: string;
    last_update?: string;
    locale: ModelsLocale;
    policy_id?: number;
    secure_config_schema: NcformSchema;
    secure_current_config: ModelsSecureModuleConfig;
    secure_default_config: ModelsSecureModuleConfig;
    state?: ModuleState;
    static_dependencies: ModelsDependencyItem[];
    status: ModuleStatus;
}

export enum ModuleStatus {
    Joined = 'joined',
    Inactive = 'inactive'
}

export interface ModelsModuleAShort {
    dynamic_dependencies: ModelsDependencyItem[];
    id?: number;
    info: ModelsModuleInfo;
    last_module_update: string;
    last_update?: string;
    locale: ModelsLocale;
    policy_id?: number;
    state?: string;
    static_dependencies: ModelsDependencyItem[];
}

export type ModelsModuleConfig = Record<string, any>;

export interface ModelsModuleInfo {
    actions: string[];
    events: string[];
    fields: string[];
    name: string;
    os: ModelsModuleInfoOS;
    system?: boolean;
    tags: string[];
    template: string;
    version: ModelsSemVersion;
}

export type ModelsModuleInfoOS = Record<string, string[]>;

export type ModelsModuleLocaleDesc = Record<string, ModelsLocaleDesc>;

export interface ModelsModuleS {
    action_config_schema: NcformSchema;
    changelog: ModelsChangelog;
    config_schema: NcformSchema;
    default_action_config: ModelsActionConfig;
    default_config: ModelsModuleConfig;
    default_event_config: ModelsEventConfig;
    event_config_schema: NcformSchema;
    fields_schema: NcformSchema;
    id?: number;
    info: ModelsModuleInfo;
    last_update?: string;
    locale: ModelsLocale;
    secure_config_schema: NcformSchema;
    secure_default_config: ModelsSecureModuleConfig;
    service_type?: string;
    state?: ModuleState;
    static_dependencies: ModelsDependencyItem[];
    tenant_id?: number;
}

export interface ModelsModuleSShort {
    changelog: ModelsChangelog;
    id?: number;
    info: ModelsModuleInfo;
    last_update?: string;
    locale: ModelsLocale;
    state?: string;
}

export interface ModelsSemVersion {
    major?: number;
    minor?: number;
    patch?: number;
}

export interface PrivateSystemModuleFile {
    data: string;
    path: string;
}

export interface PrivateSystemModuleFilePatch {
    action: 'move' | 'remove' | 'save';
    data?: string;
    newpath?: string;
    path: string;
}

export interface PrivateSystemModules {
    modules?: ModelsModuleS[];
    total?: number;
}

export interface PrivateSystemShortModules {
    modules?: ModelsModuleSShort[];
    total?: number;
}

export enum ModuleTemplate {
    Empty = 'empty',
    Generic = 'generic',
    Collector = 'collector',
    Detector = 'detector',
    Responder = 'responder'
}

export enum ModuleAction {
    Store = 'store',
    Release = 'release'
}

export interface PrivateModuleVersionPatch {
    action: ModuleAction;
    module: ModelsModuleS;
}

export interface ModelsModuleSecureParameter {
    server_only: boolean;
    value: any;
}

export enum ModuleState {
    Draft = 'draft',
    Release = 'release'
}
