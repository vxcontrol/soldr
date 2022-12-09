import { Filter } from '@soldr/shared';

export function defaultFilters(): Filter[] {
    return [
        {
            id: 'all',
            label: 'policies.Policies.PoliciesList.ListItemText.AllPolicies',
            count: 0,
            value: []
        },
        {
            id: 'without_groups',
            label: 'policies.Policies.PoliciesList.ListItemText.WithoutGroups',
            count: 0,
            value: [{ field: 'ngroups', value: 0 }]
        }
    ];
}
