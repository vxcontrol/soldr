import { Component, Input } from '@angular/core';

import { AuthStatus } from '@soldr/models';

@Component({
    selector: 'soldr-agent-auth-status',
    templateUrl: './agent-auth-status.component.html',
    styleUrls: ['./agent-auth-status.component.scss']
})
export class AgentAuthStatusComponent {
    @Input() view: 'detailed' | 'short' = 'short';
    @Input() status: AuthStatus;

    constructor() {}
}
