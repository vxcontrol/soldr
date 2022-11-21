import { Component, Input, OnChanges, SimpleChanges } from '@angular/core';
import { PopUpPlacements, ThemePalette } from '@ptsecurity/mosaic/core';

import { ModelsModuleInfoOS } from '@soldr/api';

import { OsFormatterService } from '../../services';
import { Architecture, OperationSystem, OperationSystemsList } from '../../types';

const ORDER_OS = ['windows', 'linux', 'darwin'];

@Component({
    selector: 'soldr-os',
    templateUrl: './os.component.html',
    styleUrls: ['./os.component.scss']
})
export class OsComponent implements OnChanges {
    @Input() os: OperationSystemsList | ModelsModuleInfoOS;
    @Input() hideTooltip: boolean;
    @Input() showLabels: boolean;
    @Input() isOnlyAvailable = false;

    items: { iconClass: string; text?: string; isDisable: boolean }[] = [];
    popUpPlacements = PopUpPlacements;
    themePalette = ThemePalette;
    tooltipText: string;

    constructor(private osFormatter: OsFormatterService) {}

    ngOnChanges({ os }: SimpleChanges): void {
        if (os?.currentValue) {
            this.items = ORDER_OS.map((os) => ({
                iconClass: this.getIconClass(os as OperationSystem),
                text: this.os.hasOwnProperty(os)
                    ? this.osFormatter.getText(os as OperationSystem, this.os[os as OperationSystem] as Architecture[])
                    : '',
                isDisable: !this.os.hasOwnProperty(os)
            }));
            this.items = this.isOnlyAvailable ? this.items.filter(({ isDisable }) => !isDisable) : this.items;
            this.tooltipText = this.items
                .map(({ text }) => text)
                .filter(Boolean)
                .join('\n');
        }
    }

    private getIconClass(os: OperationSystem): string {
        switch (os) {
            case OperationSystem.Windows:
                return 'soldr-icons-asset-windows_16';
            case OperationSystem.Linux:
                return 'soldr-icons-asset-linux_16';
            case OperationSystem.Darwin:
                return 'soldr-icons-asset-apple_16';
        }
    }
}
