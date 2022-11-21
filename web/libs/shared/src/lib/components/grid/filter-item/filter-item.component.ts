import { Component, Input } from '@angular/core';

@Component({
    selector: 'soldr-filter-item',
    templateUrl: './filter-item.component.html',
    styleUrls: ['./filter-item.component.scss']
})
export class FilterItemComponent {
    @Input() label: string;
    @Input() value: string | number;

    constructor() {}
}
