import { Component, Input, TemplateRef } from '@angular/core';
import { AgGridColumn } from 'ag-grid-angular';
import { CellStyle, CellStyleFunc } from 'ag-grid-community';

@Component({
    selector: 'soldr-column',
    templateUrl: './column.component.html',
    styleUrls: ['./column.component.scss']
})
export class ColumnComponent extends AgGridColumn {
    @Input() autoHeight: boolean;
    @Input() autoSize: boolean;
    @Input() cellClass: string;
    @Input() cellStyle: CellStyle | CellStyleFunc | undefined;
    @Input() default: boolean;
    @Input() displayName: string;
    @Input() filtrationField: string;
    @Input() flex: number;
    @Input() maxWidth: number;
    @Input() required: boolean;
    @Input() sortField: string;
    @Input() template: TemplateRef<any>;
    @Input() wrapText: boolean;

    constructor() {
        super();
    }
}
