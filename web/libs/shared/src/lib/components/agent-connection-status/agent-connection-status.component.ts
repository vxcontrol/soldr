import { Component, Input } from '@angular/core';

import { ConnectionStatus } from '@soldr/models';

@Component({
    selector: 'soldr-agent-connection-status',
    templateUrl: './agent-connection-status.component.html',
    styleUrls: ['./agent-connection-status.component.scss']
})
export class AgentConnectionStatusComponent {
    @Input() status: ConnectionStatus;
    @Input() hideTooltip: boolean;
    @Input() colored: boolean;

    constructor() {}
}
