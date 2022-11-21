import { Component, Input, OnChanges, SimpleChanges } from '@angular/core';
import { TranslocoService } from '@ngneat/transloco';

import { TAG_COLLECTOR, TAG_DETECTOR, TAG_PROVISIONER, TAG_RESPONDER } from '../../types';

@Component({
    selector: 'soldr-module-type',
    templateUrl: './module-type.component.html',
    styleUrls: ['./module-type.component.scss']
})
export class ModuleTypeComponent implements OnChanges {
    @Input() tags: string[];

    types: string[] = [];

    constructor(private transloco: TranslocoService) {}

    ngOnChanges({ tags }: SimpleChanges): void {
        if (tags?.currentValue) {
            this.types = this.tags
                ?.filter((tag) => [TAG_DETECTOR, TAG_PROVISIONER, TAG_RESPONDER, TAG_COLLECTOR].includes(tag))
                .map((tag) => this.getLabel(tag));
        }
    }

    private getLabel(tag: string) {
        switch (tag) {
            case TAG_DETECTOR:
                return this.transloco.translate('shared.Shared.ModuleView.Text.DetectorType');
            case TAG_COLLECTOR:
                return this.transloco.translate('shared.Shared.ModuleView.Text.CollectorType');
            case TAG_RESPONDER:
                return this.transloco.translate('shared.Shared.ModuleView.Text.ResponderType');
            case TAG_PROVISIONER:
                return this.transloco.translate('shared.Shared.ModuleView.Text.ProvisionerType');
            default:
                return '';
        }
    }
}
