import { Component, Input } from '@angular/core';

@Component({
    selector: 'soldr-breadcrumbs',
    templateUrl: './breadcrumbs.component.html',
    styleUrls: ['./breadcrumbs.component.scss']
})
export class BreadcrumbsComponent {
    @Input() items: {
        link?: string | (string | number)[];
        text: string;
        query?: Record<string, string | number>;
        disabled?: boolean;
    }[];

    constructor() {}
}
