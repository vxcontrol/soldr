/* eslint-disable @typescript-eslint/naming-convention */
import { ModelsActionConfigItem, ModelsEventConfigItem, ModelsModuleInfoOS, ModelsModuleLocaleDesc } from './modules';

export interface ModelsOptionsActions {
    config: ModelsActionConfigItem;
    locale: ModelsModuleLocaleDesc;
    module_name: string;
    module_os: ModelsModuleInfoOS;
    name: string;
}

export interface ModelsOptionsEvents {
    config: ModelsEventConfigItem;
    locale: ModelsModuleLocaleDesc;
    module_name: string;
    module_os: ModelsModuleInfoOS;
    name: string;
}

export interface ModelsOptionsFields {
    locale: ModelsModuleLocaleDesc;
    module_name: string;
    module_os: ModelsModuleInfoOS;
    name: string;
}

export interface ModelsOptionsTags {
    locale: ModelsModuleLocaleDesc;
    module_name: string;
    module_os: ModelsModuleInfoOS;
    name: string;
}

export interface PrivateOptionsActions {
    actions?: ModelsOptionsActions[];
    total?: number;
}

export interface PrivateOptionsEvents {
    events?: ModelsOptionsEvents[];
    total?: number;
}

export interface PrivateOptionsFields {
    fields?: ModelsOptionsFields[];
    total?: number;
}

export interface PrivateOptionsTags {
    tags?: ModelsOptionsTags[];
    total?: number;
}
