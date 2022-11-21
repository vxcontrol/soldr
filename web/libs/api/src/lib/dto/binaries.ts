/* eslint-disable @typescript-eslint/naming-convention */
export interface ModelsBinary {
    hash?: string;
    id?: number;
    info: ModelsBinaryInfo;
    tenant_id?: number;
    type: string;
    upload_date?: string;
    version: string;
}

export interface ModelsBinaryChksum {
    md5: string;
    sha256: string;
}

export interface ModelsBinaryInfo {
    chksums: Record<string, ModelsBinaryChksum>;
    files: string[];
    version: ModelsBinaryVersion;
}

export interface ModelsBinaryVersion {
    build?: number;
    major?: number;
    minor?: number;
    patch?: number;
    rev?: string;
}

export interface PrivateBinaries {
    binaries?: ModelsBinary[];
    total?: number;
}
