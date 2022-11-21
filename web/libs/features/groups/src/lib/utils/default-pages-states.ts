export interface GroupsPageState {
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
        policies: {
            opened: boolean;
        };
    };
}

export interface GroupPageState {
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
        policies: {
            opened: boolean;
        };
    };
    agentsTab: {
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
    };
    policyTab: {
        rightSidebar: {
            opened: boolean;
            width?: string;
            parameters: {
                opened: boolean;
            };
            groups: {
                opened: boolean;
            };
            modules: {
                opened: boolean;
            };
            tags: {
                opened: boolean;
            };
        };
    };
}

export interface GroupModuleState {
    leftSidebar: {
        opened: boolean;
        width?: string;
    };
}

export const defaultGroupsPageState = (): GroupsPageState => ({
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
        },
        policies: {
            opened: true
        }
    }
});

export const defaultGroupPageState = (): GroupPageState => ({
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
        },
        policies: {
            opened: true
        }
    },
    agentsTab: {
        rightSidebar: {
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
    },
    policyTab: {
        rightSidebar: {
            opened: true,
            parameters: {
                opened: true
            },
            groups: {
                opened: true
            },
            modules: {
                opened: true
            },
            tags: {
                opened: true
            }
        }
    }
});

export const defaultGroupModuleState = (): GroupModuleState => ({
    leftSidebar: {
        opened: true
    }
});
