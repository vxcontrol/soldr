import {
    ModelsModuleAShort,
    ModelsPolicy,
    ModelsPolicyInfo,
    PrivatePolicies,
    PrivatePolicy,
    PrivatePolicyDetails
} from '@soldr/api';
import { Architecture, OperationSystem, OperationSystemsList } from '@soldr/shared';

export interface PolicyInfo extends ModelsPolicyInfo {
    os: OperationSystemsList;
}

// eslint-disable-next-line @typescript-eslint/no-empty-interface
export interface Policy extends Omit<ModelsPolicy, 'info'> {
    details?: PrivatePolicyDetails;
    info: PolicyInfo;
    _origin: ModelsPolicy;
}

const aggregateModulesOs = (modules: ModelsModuleAShort[]) =>
    modules.reduce((acc, module) => {
        for (const os of Object.keys(module.info.os)) {
            const osKey = os as OperationSystem;
            const addedArch = module.info.os[os] as Architecture[];
            acc[osKey] = acc[osKey] || [];
            acc[osKey] = [...acc[osKey], ...addedArch.filter((arch) => !acc[osKey].includes(arch))];
        }

        return acc;
    }, {} as OperationSystemsList);

export const privatePoliciesToModels = (data: PrivatePolicies) =>
    data.policies?.map((policy) => {
        const details = data.details.find((item) => item.hash === policy.hash);

        return {
            ...policy,
            details,
            info: {
                ...policy.info,
                os: aggregateModulesOs(details.modules || [])
            },
            _origin: policy
        } as Policy;
    });

export const privatePolicyToModel = (data: PrivatePolicy) =>
    ({
        ...data.policy,
        details: data.details,
        info: {
            ...data.policy.info,
            os: aggregateModulesOs(data.details.modules || [])
        },
        _origin: data.policy
    } as Policy);

export const policyToDto = (policy: Policy) => ({
    ...policy._origin,
    info: {
        ...policy._origin.info,
        name: policy.info.name,
        tags: policy.info.tags,
        system: policy.info.system
    }
});
