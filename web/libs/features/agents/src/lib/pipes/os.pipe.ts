import { Pipe, PipeTransform } from '@angular/core';
import { TranslocoService } from '@ngneat/transloco';

import { Architecture, OperationSystem, OperationSystemsList } from '@soldr/shared';

@Pipe({
    name: 'os'
})
export class OsPipe implements PipeTransform {
    constructor(private transloco: TranslocoService) {}

    transform(value: OperationSystemsList): unknown {
        return Object.keys(value)
            .map(
                (os) =>
                    `${this.getOperationSystemText(os as OperationSystem)} ${value[os as OperationSystem]
                        .map((arch: Architecture) => this.getArchText(arch))
                        .join(', ')}`
            )
            .join(', ');
    }

    private getOperationSystemText(os: OperationSystem) {
        switch (os) {
            case OperationSystem.Windows:
                return this.transloco.translate('shared.Shared.Os.Text.Windows');
            case OperationSystem.Linux:
                return this.transloco.translate('shared.Shared.Os.Text.Linux');
            case OperationSystem.Darwin:
                return this.transloco.translate('shared.Shared.Os.Text.Macos');
        }
    }

    private getArchText(arch: Architecture) {
        switch (arch) {
            case Architecture.I386:
                return this.transloco.translate('shared.Shared.Os.Text.I386');
            case Architecture.Amd64:
                return this.transloco.translate('shared.Shared.Os.Text.Amd64');
        }
    }
}
