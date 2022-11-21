export interface PoliciesPageState {
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
        groups: {
            opened: boolean;
        };
    };
}

export interface PolicyPageState {
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
        groups: {
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
    groupsTab: {
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
    };
}

export interface PolicyModuleState {
    leftSidebar: {
        opened: boolean;
        width?: string;
    };
}

export const defaultPoliciesPageState = (): PoliciesPageState => ({
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
        },
        groups: {
            opened: true
        }
    }
});

export const defaultPolicyPageState = (): PolicyPageState => ({
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
        groups: {
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
    groupsTab: {
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
    }
});

export const defaultPolicyModuleState = (): PolicyModuleState => ({
    leftSidebar: {
        opened: true
    }
});
