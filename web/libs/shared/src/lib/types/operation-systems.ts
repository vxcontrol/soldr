import { ListItem } from '@soldr/shared';

export enum OperationSystem {
    Windows = 'windows',
    Linux = 'linux',
    Darwin = 'darwin'
}
export enum Architecture {
    Amd64 = 'amd64',
    I386 = '386'
}
export type OperationSystemsList = Record<OperationSystem, Architecture[]>;

export const osList: readonly ListItem[] = [
    { label: 'shared.Shared.Pseudo.ListItemText.WindowsI386Dot', value: 'windows:386' },
    { label: 'shared.Shared.Pseudo.ListItemText.WindowsAmd64Dot', value: 'windows:amd64' },
    { label: 'shared.Shared.Pseudo.ListItemText.LinuxI386Dot', value: 'linux:386' },
    { label: 'shared.Shared.Pseudo.ListItemText.LinuxAmd64Dot', value: 'linux:amd64' },
    { label: 'shared.Shared.Pseudo.ListItemText.DarwinAmd64Dot', value: 'darwin:amd64' }
];

export const moduleOsList: readonly ListItem[] = [
    { label: 'Shared.Pseudo.ListItemText.WindowsI386', value: 'windows:386' },
    { label: 'Shared.Pseudo.ListItemText.WindowsAmd64', value: 'windows:amd64' },
    { label: 'Shared.Pseudo.ListItemText.LinuxI386', value: 'linux:386' },
    { label: 'Shared.Pseudo.ListItemText.LinuxAmd64', value: 'linux:amd64' },
    { label: 'Shared.Pseudo.ListItemText.DarwinAmd64', value: 'darwin:amd64' }
];

export const moduleOsListGroupByOs: readonly { label: string; items: ListItem[] }[] = [
    {
        label: 'Shared.Os.Text.Windows',
        items: [
            { label: 'Shared.Pseudo.ListItemText.WindowsI386', value: 'windows:386' },
            { label: 'Shared.Pseudo.ListItemText.WindowsAmd64', value: 'windows:amd64' }
        ]
    },
    {
        label: 'Shared.Os.Text.Linux',
        items: [
            { label: 'Shared.Pseudo.ListItemText.LinuxI386', value: 'linux:386' },
            { label: 'Shared.Pseudo.ListItemText.LinuxAmd64', value: 'linux:amd64' }
        ]
    },
    {
        label: 'Shared.Os.Text.Macos',
        items: [{ label: 'Shared.Pseudo.ListItemText.DarwinAmd64', value: 'darwin:amd64' }]
    }
];
