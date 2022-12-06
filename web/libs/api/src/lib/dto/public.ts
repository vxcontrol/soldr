import { ModelsRole } from './roles';
import { ModelsService } from './services';
import { ModelsTenant } from './tenants';
import { ModelsUser } from './users';

export interface ModelsSignIn {
    mail: string;
    password: string;
    token?: string;
}

export enum Themes {
    Dark = 'dark',
    Light = 'light'
}

export interface PublicInfo {
    develop?: boolean;
    privileges?: string[];
    role?: ModelsRole;
    service?: PublicInfoService;
    services?: PublicInfoService[];
    sso: boolean;
    ui_style: Themes;
    tenant?: ModelsTenant;
    type?: string;
    user?: ModelsUser;
}

export type PublicInfoService = Omit<ModelsService, 'info'>;

export enum Permission {
    CreateAgents = 'vxapi.agents.api.create',
    DeleteAgents = 'vxapi.agents.api.delete',
    EditAgents = 'vxapi.agents.api.edit',
    ViewAgents = 'vxapi.agents.api.view',
    DownloadsAgents = 'vxapi.agents.downloads',

    CreateGroups = 'vxapi.groups.api.create',
    DeleteGroups = 'vxapi.groups.api.delete',
    EditGroups = 'vxapi.groups.api.edit',
    ViewGroups = 'vxapi.groups.api.view',

    ViewModulesEvents = 'vxapi.modules.events',
    ViewModulesOperations = 'vxapi.modules.interactive',
    CreateModules = 'vxapi.modules.api.create',
    DeleteModules = 'vxapi.modules.api.delete',
    EditModules = 'vxapi.modules.api.edit',
    ViewModules = 'vxapi.modules.api.view',
    ExportModules = 'vxapi.modules.control.export',
    ImportModules = 'vxapi.modules.control.import',
    EditSecureConfig = 'vxapi.modules.secure-config.edit',
    ViewSecureConfig = 'vxapi.modules.secure-config.view',

    CreatePolicies = 'vxapi.policies.api.create',
    DeletePolicies = 'vxapi.policies.api.delete',
    EditPolicies = 'vxapi.policies.api.edit',
    ViewPolicies = 'vxapi.policies.api.view',
    LinkPolicies = 'vxapi.policies.control.link',

    ViewRoles = 'vxapi.roles.api.view',

    CreateServices = 'vxapi.services.api.create',
    DeleteServices = 'vxapi.services.api.delete',
    EditServices = 'vxapi.services.api.edit',
    ViewServices = 'vxapi.services.api.view',

    CreateUsers = 'vxapi.users.api.create',
    DeleteUsers = 'vxapi.users.api.delete',
    EditUsers = 'vxapi.users.api.edit',
    ViewUsers = 'vxapi.users.api.view',

    SystemControlUpdate = 'vxapi.system.control.update',
    SystemLoggingControl = 'vxapi.system.logging.control',
    SystemMonitoringControl = 'vxapi.system.monitoring.control',

    CreateTemplates = 'vxapi.templates.create',
    DeleteTemplates = 'vxapi.templates.delete',
    ViewTemplates = 'vxapi.templates.view'
}
