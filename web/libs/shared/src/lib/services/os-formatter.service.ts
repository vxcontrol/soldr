import { Injectable } from '@angular/core';
import { TranslocoService } from '@ngneat/transloco';

import { Architecture, OperationSystem } from '@soldr/shared';

@Injectable({
    providedIn: 'root'
})
export class OsFormatterService {
    constructor(private transloco: TranslocoService) {}

    getText(os: OperationSystem, architectures: Architecture[]): string {
        const delimiter = ', ';

        return `${this.getOperationSystemText(os)} ${architectures
            .map((arch) => this.getArchText(arch))
            .join(delimiter)}`;
    }

    getOperationSystemText(os: OperationSystem) {
        switch (os) {
            case OperationSystem.Windows:
                return this.transloco.translate('shared.Shared.Os.Text.Windows');
            case OperationSystem.Linux:
                return this.transloco.translate('shared.Shared.Os.Text.Linux');
            case OperationSystem.Darwin:
                return this.transloco.translate('shared.Shared.Os.Text.Macos');
        }
    }

    getArchText(arch: Architecture) {
        switch (arch) {
            case Architecture.I386:
                return this.transloco.translate('shared.Shared.Os.Text.I386');
            case Architecture.Amd64:
                return this.transloco.translate('shared.Shared.Os.Text.Amd64');
        }
    }
}
