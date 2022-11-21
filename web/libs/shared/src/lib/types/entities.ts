import {
    DependencyType,
    ModelsAgentDependency,
    ModelsGroupDependency,
    ModelsModuleSShort,
    ModelsPolicy,
    ModelsPolicyDependency
} from '@soldr/api';
import { Agent, AgentModule, Group, GroupModule, Policy, PolicyModule } from '@soldr/models';

export type Entity = Agent | Group | Policy;
export type EntityModule = AgentModule | GroupModule | PolicyModule;
export type ReadOnlyModule = AgentModule | GroupModule;
export type EntityDependency = ModelsAgentDependency | ModelsGroupDependency | ModelsPolicyDependency;

export interface Dependency {
    status: boolean;
    type: DependencyType;
    moduleName?: string;
    module: ModelsModuleSShort;
    moduleLink: string;
    sourceModuleName?: string;
    sourceModule: ModelsModuleSShort;
    sourceModuleLink: string;
    policy: ModelsPolicy;
    description: string;
    minModuleVersion: string;
    minAgentVersion: string;
}
