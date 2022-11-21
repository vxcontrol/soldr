import { ModelsModuleA, PrivatePolicyModuleDetails, PrivatePolicyModules } from '@soldr/api';

export interface PolicyModule extends ModelsModuleA {
    details?: PrivatePolicyModuleDetails;
}

export const privatePoliciesModulesToModels = (data: PrivatePolicyModules) =>
    data.modules?.map(
        (module: ModelsModuleA) =>
            ({
                ...module,
                details: data.details.find((item) => item.name === module.info.name)
            } as PolicyModule)
    );
