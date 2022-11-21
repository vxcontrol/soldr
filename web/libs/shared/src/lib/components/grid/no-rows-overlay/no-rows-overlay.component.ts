import { Component, TemplateRef } from '@angular/core';
import { INoRowsOverlayAngularComp } from 'ag-grid-angular';
import { INoRowsOverlayParams } from 'ag-grid-community';

export interface NoRowsOverlayParams extends INoRowsOverlayParams {
    emptyText: string;
    isFirstLoading: boolean;
    template: TemplateRef<any>;
}

@Component({
    selector: 'soldr-no-rows-overlay',
    template: `
        <ng-container *ngIf="params.template && !params.isFirstLoading; else emptyText">
            <ng-container *ngTemplateOutlet="params.template"></ng-container>
        </ng-container>
        <ng-template #emptyText>
            <span
                *transloco="let tCommon; read: 'common'"
                class="text_second"
                [innerHTML]="params.emptyText || tCommon('Common.Pseudo.Text.NoData')"
            >
            </span>
        </ng-template>
    `
})
export class NoRowsOverlayComponent implements INoRowsOverlayAngularComp {
    params: NoRowsOverlayParams;

    agInit(params: NoRowsOverlayParams): void {
        this.params = params;
    }
}
