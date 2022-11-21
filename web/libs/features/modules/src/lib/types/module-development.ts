import { Observable } from 'rxjs';

import { PropertyType } from '@soldr/shared';

export interface ConfigurationItem {
    name: string;
    required: boolean;
    type: Omit<PropertyType, PropertyType.COMP | PropertyType.HTML | PropertyType.NONE>;
    fields: string;
}

export interface SecureConfigurationItem extends ConfigurationItem {
    serverOnly: boolean;
}

export enum EventConfigurationItemType {
    Atomic = '{"$ref":"#/definitions/events.atomic"}',
    Aggregation = '{"$ref":"#/definitions/events.aggregation"}',
    Correlation = '{"$ref":"#/definitions/events.correlation"}'
}

export interface EventConfigurationItem {
    config_fields: string;
    fields: string[];
    keys: ConfigurationItem[];
    name: string;
    type: EventConfigurationItemType;
}

export interface ActionConfigurationItem {
    config_fields: string;
    fields: string[];
    keys: ConfigurationItem[];
    name: string;
    type: string;
    priority: number;
}

export interface ChangelogVersionRecord {
    isRelease?: boolean;
    version: string;
    date: string;
    locales: {
        ru: {
            title: string;
            description: string;
        };
        en: {
            title: string;
            description: string;
        };
    };
}

export interface ModuleSection {
    validateForms: () => Observable<boolean>;
}

export enum ModuleFolderSection {
    Agent = 'cmodule',
    Browser = 'bmodule',
    Server = 'smodule'
}

export enum FileTreeNodeType {
    Section = 'section',
    Part = 'part',
    Folder = 'folder',
    File = 'file'
}

export class FilesTreeNode {
    type: FileTreeNodeType;
    level: number;
    data: {
        name: string;
        path: string;
    };
    children?: FilesTreeNode[];
    parent?: FilesTreeNode;
}
