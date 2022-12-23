import { Injectable } from '@angular/core';
import { combineLatest, filter, map, Subject } from 'rxjs';

import { Group, Policy } from '@soldr/models';
import { ConflictsByEntity, LinkPolicyToGroupFacade } from '@soldr/shared';
import { PoliciesFacade } from '@soldr/store/policies';
import { SharedFacade } from '@soldr/store/shared';

@Injectable()
export class LinkPolicyFacadeService implements LinkPolicyToGroupFacade<Group, Policy> {
    baseEntity$ = new Subject<Policy>();
    conflictedItem = new Subject<Group>();
    conflictedItem$ = this.conflictedItem.asObservable();
    disabled$ = this.baseEntity$.pipe(map((v) => !v));
    linkedIds$ = this.baseEntity$.pipe(map((entity) => entity?.details?.groups?.map(({ id }) => id) || []));
    isLoading$ = combineLatest([
        this.policiesFacade.isLinkingPolicy$,
        this.policiesFacade.isUnlinkingPolicy$,
        this.sharedFacade.isLoadingAllPolicies$
    ]).pipe(
        map(
            ([isLinkingGroup, isUnlinkingGroup, isLoadingAllPolicies]) =>
                isLinkingGroup || isUnlinkingGroup || isLoadingAllPolicies
        )
    );

    items$ = this.sharedFacade.allGroups$;
    linked$ = combineLatest([this.linkedIds$, this.items$]).pipe(
        map(([linkedIds, items]) => items.filter((item) => linkedIds.includes(item.id)))
    );
    modulesInLinked$ = this.linked$.pipe(
        map(
            (linked) =>
                Array.from(
                    linked.reduce((acc, item) => {
                        const modules = item.details?.joined_modules?.split(',');

                        return new Set([...acc, ...modules]);
                    }, new Set())
                ) as string[]
        )
    );
    notLinked$ = combineLatest([this.linkedIds$, this.items$]).pipe(
        map(([linkedIds, items]) => items.filter((item) => !linkedIds.includes(item.id)))
    );
    conflictsByEntityId$ = combineLatest([
        this.sharedFacade.allPolicies$,
        this.linked$,
        this.notLinked$,
        this.baseEntity$
    ]).pipe(
        filter(([allPolicies]) => !!allPolicies.length),
        map(([allPolicies, linked, notLinked, policy]) =>
            notLinked.reduce((acc, group) => {
                const groupModuleNames = group.details?.joined_modules?.split(',');

                for (const groupModuleName of groupModuleNames) {
                    const policyModuleNames = policy?.details?.joined_modules?.split(',');

                    if (groupModuleName && policyModuleNames?.includes(groupModuleName)) {
                        const groupModule = group.details?.modules?.find(
                            (module) => module.info.name === groupModuleName
                        );
                        const linkedGroups = linked.filter(({ hash }) => hash !== group.hash);
                        const conflictedGroup = [...linkedGroups, ...notLinked].find((item) =>
                            item.details?.joined_modules?.split(',').includes(groupModuleName)
                        );

                        if (conflictedGroup) {
                            const policiesIdsInConflictedGroup = group?.details?.policies?.map(({ id }) => id);

                            const policiesInConflictedGroup = allPolicies.filter(({ id }) =>
                                policiesIdsInConflictedGroup?.includes(id)
                            );
                            const conflictedPolicy = policiesInConflictedGroup.find((policy) =>
                                policy.details?.joined_modules?.split(',').includes(groupModuleName)
                            );

                            if (groupModule && conflictedPolicy) {
                                acc[group.id] = acc[group.id] || [];
                                acc[group.id].push({
                                    module: groupModule,
                                    conflictedPolicy
                                });
                            }
                        }
                    }
                }

                return acc;
            }, {} as ConflictsByEntity)
        )
    );
    unavailable$ = combineLatest([this.conflictsByEntityId$, this.notLinked$]).pipe(
        map(([conflictsByEntityId, notLinked]) =>
            notLinked.filter((item) => Object.keys(conflictsByEntityId[item.id] || {}).length > 0)
        )
    );
    available$ = combineLatest([this.conflictsByEntityId$, this.notLinked$]).pipe(
        map(([conflictsByEntityId, notLinked]) =>
            notLinked.filter((item) => Object.keys(conflictsByEntityId[item.id] || {}).length === 0)
        )
    );
    groupedConflictsByPolicy$ = combineLatest([this.conflictsByEntityId$, this.conflictedItem$]).pipe(
        map(([conflictsByEntityId, conflictedItem]) =>
            Array.from(
                conflictsByEntityId[conflictedItem.id].reduce((acc, conflict) => {
                    acc.set(conflict.conflictedPolicy, [
                        ...(acc.get(conflict.conflictedPolicy) || []),
                        conflict.module
                    ]);

                    return acc;
                }, new Map()),
                ([conflictedPolicy, modules]) => ({ conflictedPolicy, modules })
            )
        )
    );
    groupedConflictsByModule$ = combineLatest([this.conflictsByEntityId$, this.conflictedItem$]).pipe(
        map(([conflictsByEntityId, conflictedItem]) => conflictsByEntityId[conflictedItem.id])
    );
    conflictGroup$ = this.conflictedItem$;
    conflictPolicy$ = this.baseEntity$;

    constructor(private policiesFacade: PoliciesFacade, private sharedFacade: SharedFacade) {}

    fetchData(): void {
        this.sharedFacade.fetchAllPolicies();
        this.sharedFacade.fetchAllGroups();
    }

    link(hash: string, item: Group): void {
        this.policiesFacade.linkPolicyToGroup(hash, item._origin);
    }

    unlink(hash: string, item: Group): void {
        this.policiesFacade.unlinkPolicyFromGroup(hash, item._origin);
    }
}
