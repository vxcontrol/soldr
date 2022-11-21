import { Injectable } from '@angular/core';
import { map, combineLatest, skipWhile } from 'rxjs';

import { Dependency, getDependencyDescriptionKey } from '@soldr/shared';
import { AgentCardFacade } from '@soldr/store/agents';
import { SharedFacade } from '@soldr/store/shared';

@Injectable({
    providedIn: 'root'
})
export class AgentDependenciesFacadeService {
    $dependencies$ = combineLatest([this.agentCardFacade.agent$, this.sharedFacade.allModules$]).pipe(
        skipWhile(([, modules]) => modules.length === 0),
        map(([agent, modules]) => {
            const dependencies = agent?.details?.dependencies;

            return (
                dependencies?.map((dependency, index) => {
                    const module = modules.find((item) => item.info.name === dependency.module_name);
                    const moduleLink = `/agents/${agent?.hash}/modules/${module?.info.name}`;
                    const sourceModule = modules.find((item) => item.info.name === dependency.source_module_name);
                    const sourceModuleLink = `/agents/${agent?.hash}/modules/${sourceModule.info.name}`;
                    const policy = agent?.details?.policies?.find(({ id }) => id === dependency.policy_id);

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

    constructor(private agentCardFacade: AgentCardFacade, private sharedFacade: SharedFacade) {}
}
