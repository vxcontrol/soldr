/*
Похоже взято отсюда https://github.com/ag-grid/ag-grid-angular/issues/206#issuecomment-394106165
Аналогичный подход описан тут https://blog.angularindepth.com/easier-embedding-of-angular-ui-in-ag-grid-52db93b73884

Для чего нужна эта директива? Она кажется не используется.
*/

import { Directive, Input, OnInit, TemplateRef } from '@angular/core';

import { ColumnComponent, TemplateCellComponent } from '@soldr/shared';

@Directive({
    selector: '[soldrTemplateColumn]'
})
export class TemplateColumnDirective implements OnInit {
    @Input()
    template: TemplateRef<any>;

    constructor(private host: ColumnComponent) {}

    ngOnInit(): void {
        this.host.comparator = () => 0;
        this.host.cellRendererFramework = TemplateCellComponent;
        this.host.cellRendererParams = {
            template: this.template,
            elements: {},
            setElement(rowId: string | number, element: any) {
                this.elements[rowId] = element;
            },
            getElement(rowId: string | number): HTMLElement {
                return this.elements[rowId];
            }
        };
    }
}
