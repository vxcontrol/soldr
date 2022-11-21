import { ModelsModuleA, PrivateAgentModuleDetails, PrivateAgentModules } from '@soldr/api';

export interface AgentModule extends ModelsModuleA {
    details?: PrivateAgentModuleDetails;
}

export const privateAgentModulesToModels = (data: PrivateAgentModules) =>
    data.modules?.map(
        (module: ModelsModuleA) =>
            ({
                ...module,
                details: data.details.find((item) => item.name === module.info.name)
            } as AgentModule)
    );
