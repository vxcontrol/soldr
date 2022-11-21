import { Injectable } from '@angular/core';
import { TranslocoService } from '@ngneat/transloco';
import { ThemePalette } from '@ptsecurity/mosaic/core';
import { McModalRef, McModalService, ModalSize } from '@ptsecurity/mosaic/modal';
import { Subject } from 'rxjs';

@Injectable({ providedIn: 'root' })
export class ModalInfoService {
    tplModal: McModalRef;

    constructor(private modalService: McModalService, private transloco: TranslocoService) {}

    openErrorInfoModal(text?: string) {
        this.tplModal = this.modalService.confirm({
            mcSize: ModalSize.Small,
            mcContent: text ? text : this.transloco.translate('shared.Shared.ModalInfo.Text.Error'),
            mcCancelText: this.transloco.translate('common.Common.Pseudo.ButtonText.Close'),
            mcOnCancel: () => this.destroyTplModal()
        });
    }

    openUnsavedChangesModal(isForm?: boolean) {
        const result$ = new Subject<boolean>();

        this.tplModal = this.modalService.create({
            mcSize: ModalSize.Normal,
            mcContent: isForm
                ? this.transloco.translate('shared.Shared.ModalInfo.Text.LeaveFormOnUnsavedChanges')
                : this.transloco.translate('shared.Shared.ModalInfo.Text.LeavePageOnUnsavedChanges'),
            mcFooter: [
                {
                    label: this.transloco.translate('shared.Shared.ModalInfo.ButtonText.ExitWithoutSaving'),
                    type: ThemePalette.Primary,
                    onClick: () => {
                        result$.next(true);
                        this.destroyTplModal();
                    }
                },
                {
                    autoFocus: true,
                    label: this.transloco.translate('shared.Shared.ModalInfo.ButtonText.Stay'),
                    type: ThemePalette.Default,
                    onClick: () => {
                        result$.next(false);
                        this.destroyTplModal();
                    }
                }
            ],
            mcClosable: false
        });

        return result$;
    }

    destroyTplModal() {
        this.tplModal.destroy();
    }
}
