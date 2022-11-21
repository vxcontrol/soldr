/* eslint-disable @typescript-eslint/naming-convention */
import { ModelsRole } from './roles';
import { ModelsTenant } from './tenants';

export interface ModelsUser {
    hash?: string;
    id?: number;
    mail: string;
    name: string;
    role_id?: number;
    status: string;
    tenant_id?: number;
}

export interface ModelsPassword {
    confirm_password?: string;
    current_password: string;
    password: string;
}

export interface ModelsUserPassword {
    hash?: string;
    id?: number;
    mail: string;
    name: string;
    password: string;
    role_id?: number;
    status: string;
    tenant_id?: number;
}

export interface ModelsUserRoleTenant {
    hash?: string;
    id?: number;
    mail: string;
    name: string;
    role?: ModelsRole;
    role_id?: number;
    status: string;
    tenant?: ModelsTenant;
    tenant_id?: number;
}

export interface PrivateUsers {
    total?: number;
    users?: ModelsUserRoleTenant[];
}
