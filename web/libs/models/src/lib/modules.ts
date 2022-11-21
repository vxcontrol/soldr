import { ModelsModuleS, PrivateSystemModules } from '@soldr/api';

export interface Module extends ModelsModuleS {
    _origin: ModelsModuleS;
}

export interface FilesContent {
    [filename: string]: { content: string; loaded: boolean };
}

export const manyModulesToModels = (data: PrivateSystemModules) =>
    data.modules?.map(
        (module: ModelsModuleS) =>
            ({
                ...module,
                _origin: module
            } as Module)
    );
