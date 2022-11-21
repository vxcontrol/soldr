import { GridColumnFilterItem } from '@soldr/shared';

export function gridFilters(): { [field: string]: GridColumnFilterItem[] } {
    return {
        status: [
            {
                label: 'agents.Agents.AgentsList.DropdownItemText.Connected',
                value: 'connected'
            },
            {
                label: 'agents.Agents.AgentsList.DropdownItemText.Disconnected',
                value: 'disconnected'
            },
            {
                label: 'shared.Shared.Pseudo.DropdownItemText.Any',
                value: undefined
            }
        ],
        // eslint-disable-next-line @typescript-eslint/naming-convention
        auth_status: [
            {
                label: 'agents.Agents.AgentsList.DropdownItemText.Authorized',
                value: 'authorized'
            },
            {
                label: 'agents.Agents.AgentsList.DropdownItemText.Unauthorized',
                value: 'unauthorized'
            },
            {
                label: 'agents.Agents.AgentsList.DropdownItemText.Blocked',
                value: 'blocked'
            },
            {
                label: 'shared.Shared.Pseudo.DropdownItemText.Any',
                value: undefined
            }
        ],
        versions: [],
        modules: [],
        groups: [],
        tags: []
    };
}
