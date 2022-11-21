export interface AgentsPageState {
    leftSidebar: {
        opened: boolean;
        width?: string;
    };

    rightSidebar: {
        opened: boolean;
        width?: string;
        parameters: {
            opened: boolean;
        };
        modules: {
            opened: boolean;
        };
        tags: {
            opened: boolean;
        };
    };
}

export interface AgentPageState {
    leftSidebar: {
        opened: boolean;
        width?: string;
        parameters: {
            opened: boolean;
        };
        modules: {
            opened: boolean;
        };
        tags: {
            opened: boolean;
        };
    };
}

export interface AgentModuleState {
    leftSidebar: {
        opened: boolean;
        width?: string;
    };
}

export const defaultAgentsPageState = (): AgentsPageState => ({
    leftSidebar: {
        opened: true
    },
    rightSidebar: {
        opened: true,
        parameters: {
            opened: true
        },
        tags: {
            opened: true
        },
        modules: {
            opened: true
        }
    }
});

export const defaultAgentPageState = (): AgentPageState => ({
    leftSidebar: {
        opened: true,
        parameters: {
            opened: true
        },
        modules: {
            opened: true
        },
        tags: {
            opened: true
        }
    }
});

export const defaultAgentModuleState = (): AgentModuleState => ({
    leftSidebar: {
        opened: true
    }
});
