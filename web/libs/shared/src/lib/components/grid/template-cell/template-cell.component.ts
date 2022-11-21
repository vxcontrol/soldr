/*
Похоже взято отсюда https://github.com/ag-grid/ag-grid-angular/issues/206#issuecomment-394106165
Аналогичный подход описан тут https://blog.angularindepth.com/easier-embedding-of-angular-ui-in-ag-grid-52db93b73884
*/

import { ChangeDetectionStrategy, Component, ElementRef, Input, OnInit, TemplateRef } from '@angular/core';
import { ICellRendererAngularComp } from 'ag-grid-angular';
import { ICellRendererParams } from 'ag-grid-community';

@Component({
    selector: 'soldr-template-cell',
    template: `
        <ng-container *ngIf="template; else defaultTemplate">
            <ng-container *ngTemplateOutlet="template; context: templateContext"></ng-container>
        </ng-container>
        <ng-template #defaultTemplate>
            <span soldrTextOverflow>{{ params.value }}</span>
        </ng-template>
    `,
    styles: [
        `
            :host {
                display: block;
                overflow: hidden;
                text-overflow: ellipsis;
                width: 100%;
                height: 100%;
                text-align: left;
            }
        `
    ],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class TemplateCellComponent implements OnInit, ICellRendererAngularComp {
    @Input()
    template: TemplateRef<any>;

    params: ICellRendererParams;

    get templateContext() {
        return {
            $implicit: this.params.value,
            params: this.params
        };
    }

    constructor(public element: ElementRef) {}

    ngOnInit(): void {
        if (this.params && (this.params as any).setElement) {
            (this.params as any).setElement(this.params.node.id, this.element.nativeElement);
        }
    }

    agInit(params: ICellRendererParams): void {
        this.params = params;
        this.template = (params as any).template as TemplateRef<any>;
    }

    refresh(params: any): boolean {
        return false;
    }
}
