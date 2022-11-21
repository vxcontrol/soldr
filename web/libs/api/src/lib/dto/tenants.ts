export interface ModelsTenant {
    description?: string;
    hash: string;
    id?: number;
    status: string;
}

export interface PrivateTenants {
    tenants?: ModelsTenant[];
    total?: number;
}
