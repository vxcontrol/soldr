import { Component, HostBinding, Input } from '@angular/core';

const LOW_LEVEL = 33;
const MEDIUM_LEVEL = 65;

enum ActionPriorityLevel {
    High = 'high',
    Medium = 'medium',
    Low = 'low'
}

@Component({
    selector: 'soldr-action-priority',
    templateUrl: './action-priority.component.html',
    styleUrls: ['./action-priority.component.scss']
})
export class ActionPriorityComponent {
    @Input() priority: number;
    @Input() mini: boolean;

    actionPriorityLevel = ActionPriorityLevel;

    @HostBinding('class') get class() {
        return this.priority > LOW_LEVEL && this.priority <= MEDIUM_LEVEL
            ? 'action-priority_medium'
            : this.priority > MEDIUM_LEVEL
            ? 'action-priority_high'
            : 'action-priority_low';
    }

    get level() {
        return this.priority > LOW_LEVEL && this.priority <= MEDIUM_LEVEL
            ? ActionPriorityLevel.Medium
            : this.priority > MEDIUM_LEVEL
            ? ActionPriorityLevel.High
            : ActionPriorityLevel.Low;
    }

    constructor() {}
}
