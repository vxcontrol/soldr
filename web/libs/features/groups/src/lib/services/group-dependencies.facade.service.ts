import { Injectable } from '@angular/core';
import { combineLatest, map, skipWhile } from 'rxjs';

import { Dependency, getDependencyDescriptionKey } from '@soldr/shared';
import { GroupsFacade } from '@soldr/store/groups';
import { SharedFacade } from '@soldr/store/shared';

@Injectable({
    providedIn: 'root'
})
export class GroupDependenciesFacadeService {
    dependencies$ = combineLatest([this.groupsFacade.group$, this.sharedFacade.allModules$]).pipe(
        skipWhile(([, modules]) => modules.length === 0),
        map(([group, modules]) => {
            const dependencies = group?.details?.dependencies;

            return (
                dependencies?.map((dependency, index) => {
                    const module = modules.find((item) => item.info.name === dependency.module_name);
                    const moduleLink = `/groups/${group?.hash}/modules/${module?.info.name}`;
                    const sourceModule = modules.find((item) => item.info.name === dependency.source_module_name);
                    const sourceModuleLink = `/groups/${group?.hash}/modules/${sourceModule.info.name}`;
                    const policy = group?.details?.policies?.find(({ id }) => id === dependency.policy_id);

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

    constructor(private groupsFacade: GroupsFacade, private sharedFacade: SharedFacade) {}
}
