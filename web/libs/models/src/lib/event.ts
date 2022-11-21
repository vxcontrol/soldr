import { ModelsAgent, ModelsEvent, ModelsGroup, ModelsModuleAShort, ModelsPolicy, PrivateEvents } from '@soldr/api';

export interface Event extends ModelsEvent {
    agent?: ModelsAgent;
    module?: ModelsModuleAShort;
    policy?: ModelsPolicy;
    group?: ModelsGroup;
}

export const privateEventsToModels = (data: PrivateEvents) =>
    data.events.map((event: ModelsEvent) => {
        const agent = data.agents?.find(({ id }) => id === event.agent_id);
        const module = data.modules?.find(({ id }) => id === event.module_id);
        const policy = data.policies?.find(({ id }) => id === module?.policy_id);
        const group = data.groups?.find(({ id }) => id === agent?.group_id);

        return {
            ...event,
            agent,
            group,
            module,
            policy
        } as Event;
    });
