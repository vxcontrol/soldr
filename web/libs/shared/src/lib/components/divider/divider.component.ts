import { Component, HostBinding } from '@angular/core';

@Component({
    selector: 'soldr-divider',
    template: '',
    styleUrls: ['divider.component.scss']
})
export class DividerComponent {
    @HostBinding('class.soldr-divider') soldrDivider: boolean;

    constructor() {
        this.soldrDivider = true;
    }
}
