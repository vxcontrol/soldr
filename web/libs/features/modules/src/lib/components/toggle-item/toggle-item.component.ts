import { ChangeDetectorRef, Component, EventEmitter, Host, Input, Output } from '@angular/core';

import { ToggleListComponent } from '../toggle-list/toggle-list.component';

@Component({
    selector: 'soldr-toggle-item',
    templateUrl: './toggle-item.component.html',
    styleUrls: ['./toggle-item.component.scss']
})
export class ToggleItemComponent {
    @Input() title: string;
    @Input() isExpanded = false;
    @Input() canDelete = true;

    @Output() delete = new EventEmitter();

    constructor(@Host() private list: ToggleListComponent, private cdr: ChangeDetectorRef) {}

    toggle() {
        this.isExpanded = !this.isExpanded;
        this.cdr.detectChanges();
    }
}
