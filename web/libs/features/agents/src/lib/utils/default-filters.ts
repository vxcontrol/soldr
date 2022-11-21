import { Filter } from '@soldr/shared';

export function defaultFilters(): Filter[] {
    return [
        {
            id: 'all',
            label: 'agents.Agents.AgentsList.ListItemText.AllAgents',
            count: 0,
            value: []
        },
        {
            id: 'authorized',
            label: 'agents.Agents.AgentsList.ListItemText.Authorized',
            count: 0,
            value: [{ field: 'auth_status', value: ['authorized'] }]
        },
        {
            id: 'without_groups',
            label: 'agents.Agents.AgentsList.ListItemText.WithoutGroups',
            count: 0,
            value: [
                { field: 'auth_status', value: ['authorized'] },
                { field: 'group_id', value: 0 }
            ]
        },
        {
            id: 'unauthorized',
            label: 'agents.Agents.AgentsList.ListItemText.Unauthorized',
            count: 0,
            value: [
                {
                    field: 'auth_status',
                    value: ['unauthorized']
                }
            ]
        },
        {
            id: 'blocked',
            label: 'agents.Agents.AgentsList.ListItemText.Blocked',
            count: 0,
            value: [
                {
                    field: 'auth_status',
                    value: ['blocked']
                }
            ]
        }
    ];
}
