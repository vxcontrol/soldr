import { Component, EventEmitter, Input, Output } from '@angular/core';

@Component({
    selector: 'soldr-accordion',
    templateUrl: './accordion.component.html',
    styleUrls: ['./accordion.component.scss']
})
export class AccordionComponent {
    @Input() expanded: boolean;
    @Input() title: string;
    @Input() large: boolean;
    @Input() onlyHide: boolean;
    @Input() clip = true;

    @Output() stateChanged = new EventEmitter<boolean>();

    constructor() {}

    toggle() {
        this.expanded = !this.expanded;
        this.stateChanged.emit(this.expanded);
    }
}
