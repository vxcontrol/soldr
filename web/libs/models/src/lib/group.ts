import { ModelsGroup, PrivateGroup, PrivateGroupDetails, PrivateGroups } from '@soldr/api';

export interface Group extends ModelsGroup {
    details?: PrivateGroupDetails;
    _origin: ModelsGroup;
}

export const privateGroupsToModels = (data: PrivateGroups) =>
    data.groups?.map(
        (group: ModelsGroup) =>
            ({
                ...group,
                details: data.details.find((item) => item.hash === group.hash),
                _origin: group
            } as Group)
    );

export const privateGroupToModel = (data: PrivateGroup) => ({
    ...data.group,
    details: data?.details,
    _origin: data.group
});
export const modelsGroupToModel = (group: ModelsGroup) => ({ ...group, _origin: group });

export const groupToDto = (group: Group) => ({
    ...group._origin,
    info: {
        ...group._origin.info,
        name: group.info.name,
        tags: group.info.tags
    }
});
