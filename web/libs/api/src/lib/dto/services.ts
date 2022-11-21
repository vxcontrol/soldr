/* eslint-disable @typescript-eslint/naming-convention */
export interface ModelsService {
    hash: string;
    id?: number;
    info?: ModelsServiceInfo;
    name: string;
    status: ServiceStatus;
    tenant_id?: number;
    type: string;
}

export interface ModelsServiceInfo {
    db: ModelsServiceInfoDB;
    s3: ModelsServiceInfoS3;
    server: ModelsServiceInfoServer;
}

export interface ModelsServiceInfoDB {
    host: string;
    name: string;
    pass: string;
    port: number;
    user: string;
}

export interface ModelsServiceInfoS3 {
    access_key: string;
    bucket_name: string;
    endpoint: string;
    secret_key: string;
}

export interface ModelsServiceInfoServer {
    host: string;
    port: number;
    proto: string;
}

export interface PrivateServices {
    services?: ModelsService[];
    total?: number;
}

export enum ServiceStatus {
    Created = 'created',
    Active = 'active',
    Blocked = 'blocked',
    Removed = 'removed'
}
