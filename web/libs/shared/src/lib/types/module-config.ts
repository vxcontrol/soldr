import { ModelsEventConfigAction } from '@soldr/api';

export interface EventItem {
    hasParams: boolean;
    actions: LocalizedAction[];
    localizedName: string;
    name: string;
    schema: any;
}

export interface EventDetailsItem extends EventItem {
    fields: LocalizedField[];
    localizedDescription: string;
    model: any;
    type: string;
}

export interface LocalizedField {
    localizedName: string;
    localizedDescription: string;
}

export interface LocalizedAction extends ModelsEventConfigAction {
    localizedName: string;
    localizedDescription: string;
}
