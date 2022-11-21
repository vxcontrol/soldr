import { Injectable } from '@angular/core';
import { TranslocoService } from '@ngneat/transloco';
import { McModalRef, McModalService, ModalSize } from '@ptsecurity/mosaic/modal';
import { take } from 'rxjs';

@Injectable({
    providedIn: 'root'
})
export class DialogsService {
    constructor(private modalService: McModalService, private transloco: TranslocoService) {}

    showRemoveDialog(isAll = false) {
        const confirmModalRef: McModalRef<any> = this.modalService.confirm({
            mcSize: ModalSize.Small,
            mcContent: isAll
                ? this.transloco.translate('modules.Modules.ModuleEdit.Text.RemoveAllElements')
                : this.transloco.translate('modules.Modules.ModuleEdit.Text.RemoveElement'),
            mcOkText: this.transloco.translate('common.Common.Pseudo.ButtonText.Delete'),
            mcCancelText: this.transloco.translate('common.Common.Pseudo.ButtonText.Cancel'),
            mcOnOk: () => confirmModalRef.close(true),
            mcOnCancel: () => confirmModalRef.close(false)
        });

        return confirmModalRef.afterClose.pipe(take(1));
    }
}
