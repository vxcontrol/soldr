import { Permission } from '@soldr/api';
import { PageUrls } from '@soldr/core';

export type ProxyPermission = {
    [key in keyof typeof Permission]: boolean;
};

export type ProxyPermissionPage = {
    [key in keyof typeof PageUrls]: boolean;
};
