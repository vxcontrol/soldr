import { Injectable } from '@angular/core';
import { combineLatest, map, Subject } from 'rxjs';

import { Group, Policy } from '@soldr/models';
import { ConflictsByEntity, LinkPolicyToGroupFacade } from '@soldr/shared';
import { GroupsFacade } from '@soldr/store/groups';
import { SharedFacade } from '@soldr/store/shared';

@Injectable()
export class LinkPolicyFacadeService implements LinkPolicyToGroupFacade<Policy, Group> {
    baseEntity$ = new Subject<Group>();
    conflictedItem = new Subject<Policy>();
    conflictedItem$ = this.conflictedItem.asObservable();
    disabled$ = this.baseEntity$.pipe(map((v) => !v));
    linkedIds$ = this.baseEntity$.pipe(map((entity) => entity?.details?.policies?.map(({ id }) => id) || []));
    isLoading$ = combineLatest([
        this.groupsFacade.isLinkingGroup$,
        this.groupsFacade.isUnlinkingGroup$,
        this.sharedFacade.isLoadingAllPolicies$
    ]).pipe(
        map(
            ([isLinkingGroup, isUnlinkingGroup, isLoadingAllPolicies]) =>
                isLinkingGroup || isUnlinkingGroup || isLoadingAllPolicies
        )
    );

    items$ = this.sharedFacade.allPolicies$;
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
    conflictsByEntityId$ = combineLatest([this.linked$, this.notLinked$, this.modulesInLinked$]).pipe(
        map(([linked, notLinked, modulesInLinked]) =>
            notLinked.reduce((acc, policy) => {
                const modulesNames = policy.details?.joined_modules?.split(',');

                for (const moduleName of modulesNames) {
                    if (moduleName && modulesInLinked.includes(moduleName)) {
                        const module = policy.details?.modules?.find((module) => module.info.name === moduleName);
                        const conflictedPolicy = linked.find((linkedItem) =>
                            linkedItem.details?.joined_modules.split(',').includes(moduleName)
                        );

                        acc[policy.id] = acc[policy.id] || [];
                        acc[policy.id].push({
                            module,
                            conflictedPolicy
                        });
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
                conflictsByEntityId[conflictedItem.id]?.reduce((acc, conflict) => {
                    acc.set(conflict.conflictedPolicy, [
                        ...(acc.get(conflict.conflictedPolicy) || []),
                        conflict.module
                    ]);

                    return acc;
                }, new Map()) || new Map(),
                ([conflictedPolicy, modules]) => ({ conflictedPolicy, modules })
            )
        )
    );
    groupedConflictsByModule$ = combineLatest([this.conflictsByEntityId$, this.conflictedItem$]).pipe(
        map(([conflictsByEntityId, conflictedItem]) => conflictsByEntityId[conflictedItem.id])
    );
    conflictGroup$ = this.baseEntity$;
    conflictPolicy$ = this.conflictedItem$;

    constructor(private groupsFacade: GroupsFacade, private sharedFacade: SharedFacade) {}

    fetchData(): void {
        this.sharedFacade.fetchAllPolicies();
    }

    link(hash: string, item: Policy): void {
        this.groupsFacade.linkGroupToPolicy(hash, item._origin);
    }

    unlink(hash: string, item: Policy): void {
        this.groupsFacade.unlinkGroupFromPolicy(hash, item._origin);
    }
}
