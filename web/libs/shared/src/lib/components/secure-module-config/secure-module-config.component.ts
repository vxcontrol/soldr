import { Component, Inject, Input, OnChanges, SimpleChanges, TemplateRef, ViewChild } from '@angular/core';
import { TranslocoService } from '@ngneat/transloco';
import { ThemePalette } from '@ptsecurity/mosaic/core';
import { McModalRef, McModalService, ModalSize } from '@ptsecurity/mosaic/modal';
import { MC_TOAST_CONFIG, McToastPosition, McToastService } from '@ptsecurity/mosaic/toast';
import { catchError, forkJoin, Observable, of, Subject, switchMap, take } from 'rxjs';

import { PoliciesService } from '@soldr/api';
import { PERMISSIONS_TOKEN } from '@soldr/core';

import { LanguageService } from '../../services';
import { EntityModule, NcFormProperty, NcformSchema, PropertyType, ProxyPermission } from '../../types';
import { clone } from '../../utils';
import { NcformWrapperApi, NcformWrapperComponent } from '../ncform-wrapper/ncform-wrapper.component';

interface SecureParam {
    isComplexType: boolean;
    isFetchValueForView: boolean;
    isShowedValue: boolean;
    localizedTitle: string;
    model: any;
    name: string;
    required: boolean;
    schema: NcFormProperty;
    type: PropertyType;
}

@Component({
    selector: 'soldr-secure-module-config',
    templateUrl: './secure-module-config.component.html',
    styleUrls: ['./secure-module-config.component.scss'],
    providers: [
        McToastService,
        {
            provide: MC_TOAST_CONFIG,
            useValue: {
                position: McToastPosition.BOTTOM_RIGHT,
                duration: 5000,
                delay: 2000,
                onTop: true
            }
        }
    ]
})
export class SecureModuleConfigComponent implements OnChanges {
    @Input() module: EntityModule;
    @Input() isReadOnly: boolean;
    @Input() policyHash: string;

    api: NcformWrapperApi;
    currentParam: SecureParam;
    isSaving = false;
    loadingForEditStatuses: Record<string, boolean> = {};
    modal: McModalRef;
    params: SecureParam[] = [];
    PropertyType = PropertyType;
    themePalette = ThemePalette;

    @ViewChild('editParamModalBody') modalBodyTemplate: TemplateRef<any>;
    @ViewChild('editParamModalFooter') modalFooterTemplate: TemplateRef<any>;
    @ViewChild('editComplexParam') editComplexParamNcform: TemplateRef<NcformWrapperComponent>;

    constructor(
        private transloco: TranslocoService,
        private languageService: LanguageService,
        private modalService: McModalService,
        private policiesService: PoliciesService,
        private toastService: McToastService,
        @Inject(PERMISSIONS_TOKEN) public permitted: ProxyPermission
    ) {}

    ngOnChanges({ module }: SimpleChanges): void {
        if (module?.currentValue) {
            this.params = Object.keys(this.module.secure_current_config || {}).map((name) => {
                const type = this.getValueType(name);
                const localizedTitle = this.module.locale.secure_config[name][this.languageService.lang].title;
                const originalSchema = this.module.secure_config_schema.properties[name]?.properties.value;
                const schema =
                    type === PropertyType.ARRAY
                        ? this.normalizeParamSchemaForArray(originalSchema, localizedTitle)
                        : originalSchema;

                return {
                    name,
                    localizedTitle,
                    isComplexType: [PropertyType.ARRAY, PropertyType.OBJECT].includes(type),
                    isFetchValueForView: false,
                    schema,
                    type,
                    required: this.module.secure_config_schema.required.includes(name)
                } as SecureParam;
            });
        }
    }

    show(param: SecureParam) {
        param.isFetchValueForView = true;

        this.loadValue(param).subscribe((response) => {
            param.isFetchValueForView = false;
            param.isShowedValue = true;

            if (this.getValueType(param.name) === PropertyType.ARRAY) {
                param.model = { items: response.data[param.name] };
            } else {
                param.model = response.data[param.name];
            }
        });
    }

    edit(param: SecureParam) {
        if (this.permitted.ViewSecureConfig) {
            this.loadingForEditStatuses[param.name] = true;

            this.loadValue(param).subscribe((response: any) => {
                this.initParamValue(param, response.data[param.name]);
                this.openEditModal();
                this.loadingForEditStatuses[param.name] = false;
            });
        } else {
            setTimeout(() => {
                this.initParamValue(param, '');
                this.openEditModal();
            });
        }
    }

    save(param: SecureParam) {
        const data$ = new Subject();

        this.isSaving = true;

        data$
            .pipe(
                take(1),
                switchMap((data) =>
                    forkJoin([
                        of(data),
                        this.policiesService.updateSecureParams(this.policyHash, this.module.info.name, {
                            [param.name]: data
                        })
                    ])
                ),
                catchError(() => {
                    this.isSaving = false;

                    this.toastService.show({
                        style: 'error',
                        title: this.transloco.translate(
                            'shared.Shared.ModuleConfig.ToastText.ErrorOnSavingSecureConfig'
                        ),
                        hasDismiss: false
                    });

                    return [];
                })
            )
            .subscribe(([data]) => {
                if (data === undefined) {
                    return;
                }

                const paramForUpdate = this.params.find((item) => item.name === param.name);

                paramForUpdate.model = paramForUpdate.type === PropertyType.ARRAY ? { items: data } : data;
                this.isSaving = false;

                this.toastService.show({
                    style: 'success',
                    title: this.transloco.translate('shared.Shared.ModuleConfig.ToastText.AboutSecureParamsOnEditing'),
                    hasDismiss: false
                });
                this.close();
            });

        if (param.isComplexType) {
            this.api.validate().then(({ result }) => {
                if (result) {
                    const data: any = this.api.getValue();
                    data$.next(this.castValueToType(data, param.type));
                }
            });
        } else {
            data$.next(this.castValueToType(param.model, param.type));
        }
    }

    close() {
        this.modal.close();
    }

    onRegisterApi(api: NcformWrapperApi) {
        this.api = api;
    }

    private loadValue(param: SecureParam) {
        return this.policiesService.getSecureParam(
            this.policyHash,
            this.module.info.name,
            param.name
        ) as Observable<any>;
    }

    private getValueType(name: string) {
        return this.module.secure_config_schema.properties[name]?.properties.value.type;
    }

    private castValueToType(value: any, type: PropertyType) {
        switch (type) {
            case PropertyType.ARRAY:
                return value.items;
            case PropertyType.INTEGER:
                return Number.parseInt(value as string);
            case PropertyType.NUMBER:
                return Number.parseFloat(value as string);
            case PropertyType.OBJECT:
                return value;
            default:
                return value;
        }
    }

    private normalizeParamSchemaForArray(originalSchema: NcFormProperty, title: string): NcformSchema {
        return {
            type: 'object',
            properties: {
                items: {
                    ...originalSchema,
                    ui: {
                        label: title,
                        legend: title
                    }
                }
            },
            ui: {
                showLegend: false,
                showLabel: false
            }
        } as NcformSchema;
    }

    private initParamValue(param: SecureParam, value: any) {
        if (this.getValueType(param.name) === PropertyType.ARRAY) {
            param.model = { items: value };
        } else {
            param.model = value;
        }
        this.currentParam = clone(param);
    }

    private openEditModal() {
        this.modal = this.modalService.create({
            mcSize: ModalSize.Normal,
            mcTitle: this.transloco.translate('shared.Shared.ModuleConfig.ModalTitle.ChangeSecureParam'),
            mcContent: this.modalBodyTemplate,
            mcFooter: this.modalFooterTemplate,
            mcOkText: this.transloco.translate('common.Common.Pseudo.ButtonText.Save'),
            mcCancelText: this.transloco.translate('common.Common.Pseudo.ButtonText.Cancel')
        });

        this.modal.afterClose.pipe(take(1)).subscribe(() => {
            this.currentParam = undefined;
        });
    }
}
