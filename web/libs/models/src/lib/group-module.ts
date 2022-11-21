import { ModelsModuleA, PrivateGroupModuleDetails, PrivateGroupModules } from '@soldr/api';

export interface GroupModule extends ModelsModuleA {
    details?: PrivateGroupModuleDetails;
}

export const privateGroupModulesToModels = (data: PrivateGroupModules) =>
    data.modules?.map(
        (module: ModelsModuleA) =>
            ({
                ...module,
                details: data.details.find((item) => item.name === module.info.name)
            } as GroupModule)
    );
