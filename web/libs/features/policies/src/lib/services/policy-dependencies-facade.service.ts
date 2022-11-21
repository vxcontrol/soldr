import { Injectable } from '@angular/core';
import { map, combineLatest, skipWhile } from 'rxjs';

import { Dependency, getDependencyDescriptionKey } from '@soldr/shared';
import { PoliciesFacade } from '@soldr/store/policies';
import { SharedFacade } from '@soldr/store/shared';

@Injectable({
    providedIn: 'root'
})
export class PolicyDependenciesFacadeService {
    $dependencies$ = combineLatest([this.policiesFacade.policy$, this.sharedFacade.allModules$]).pipe(
        skipWhile(([, modules]) => modules.length === 0),
        map(([policy, modules]) => {
            const dependencies = policy?.details?.dependencies;

            return (
                dependencies?.map((dependency, index) => {
                    const module = modules.find((item) => item.info.name === dependency.module_name);
                    const moduleLink = `/policies/${policy?.hash}/modules/${module?.info.name}`;
                    const sourceModule = modules.find((item) => item.info.name === dependency.source_module_name);
                    const sourceModuleLink = `/policies/${policy?.hash}/modules/${sourceModule?.info.name}`;

                    return {
                        id: index,
                        status: dependency.status,
                        type: dependency.type,
                        moduleName: dependency.module_name,
                        module,
                        moduleLink,
                        sourceModuleName: dependency.source_module_name,
                        sourceModule,
                        sourceModuleLink,
                        policy,
                        minModuleVersion: dependency.min_module_version,
                        minAgentVersion: dependency.min_agent_version,
                        description: getDependencyDescriptionKey(dependency)
                    } as Dependency;
                }) || []
            );
        })
    );

    constructor(private policiesFacade: PoliciesFacade, private sharedFacade: SharedFacade) {}
}
