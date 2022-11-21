import { animate, state, style, transition, trigger } from '@angular/animations';
import { Component, EventEmitter, Input, Output, TemplateRef, ViewChild } from '@angular/core';
import { ThemePalette } from '@ptsecurity/mosaic/core';
import { McSidepanelService } from '@ptsecurity/mosaic/sidepanel';
import { take } from 'rxjs';

import { LanguageService } from '../../services';
import { EntityModule } from '../../types';
import { clone, getActionParamsSchema, localizeSchemaParams } from '../../utils';
import { NcformWrapperApi } from '../ncform-wrapper/ncform-wrapper.component';

@Component({
    selector: 'soldr-action-details-panel',
    templateUrl: './action-details-panel.component.html',
    styleUrls: ['./action-details-panel.component.scss'],
    animations: [
        trigger('showHeaderOnScroll', [
            state(
                'showed',
                style({
                    zIndex: 99,
                    transform: 'translateY(0%)'
                })
            ),
            state(
                'hidden',
                style({
                    pointerEvents: 'none',
                    transform: 'translateY(-100%)'
                })
            ),
            transition('showed => hidden', [animate('0.2s ease-out')]),
            transition('hidden => showed', [animate('0.2s ease')])
        ])
    ]
})
export class ActionDetailsPanelComponent {
    @Input() module: EntityModule;
    @Input() isReadOnly: boolean;

    @Output() saveActionConfig = new EventEmitter<any>();

    @ViewChild('sidePanel') sidePanel: TemplateRef<any>;

    scrolled = false;
    themePalette = ThemePalette;
    actionName: string;
    action: any;
    ncformWrapperApi: NcformWrapperApi;

    private panelId = 'action-details-panel';

    constructor(private languageService: LanguageService, private sidePanelService: McSidepanelService) {}

    open(actionName: string) {
        this.actionName = actionName;

        const lang = this.languageService.lang;
        const locale = this.module.locale.actions[this.actionName][lang];
        const paramsSchema = localizeSchemaParams(
            getActionParamsSchema(this.module, actionName),
            this.module.locale.action_config[actionName],
            this.languageService.lang
        );
        const paramsModel = this.module.current_action_config[actionName];

        this.action = {
            localizedName: locale.title,
            localizedDescription: locale.description,
            priority: this.module.default_action_config[this.actionName].priority,
            fields: this.module.default_action_config[this.actionName].fields.map((fieldName) => {
                const fieldLocale = this.module.locale.fields[fieldName][lang];

                return {
                    localizedName: fieldLocale.title,
                    localizedDescription: fieldLocale.description
                };
            }),
            paramsSchema,
            paramsModel,
            paramsCount: Object.keys(paramsSchema.properties || {}).length
        };

        const ref = this.sidePanelService.open(this.sidePanel, {
            id: this.panelId
        });

        ref.afterClosed()
            .pipe(take(1))
            .subscribe(() => {
                this.scrolled = false;
            });
    }

    save() {
        this.ncformWrapperApi.validate().then(({ result }) => {
            if (result) {
                const module = clone(this.module) as EntityModule;
                module.current_action_config[this.actionName] = {
                    ...module.current_action_config[this.actionName],
                    ...this.ncformWrapperApi.getValue()
                };

                this.saveActionConfig.emit(module);
                this.sidePanelService.getSidepanelById(this.panelId).close();
            }
        });
    }

    cancel() {
        this.sidePanelService.getSidepanelById(this.panelId).close();
    }

    resetToDefault() {
        this.action.paramsModel = this.module.default_action_config[this.actionName];
    }

    onScrollBody($event: Event) {
        const headerHeight = 56;
        this.scrolled = ($event.target as HTMLElement).scrollTop > headerHeight;
    }

    onRegisterFormApi(ncformWrapperApi: NcformWrapperApi) {
        this.ncformWrapperApi = ncformWrapperApi;
    }

    get canSave() {
        return this.action?.paramsCount > 0 && this.ncformWrapperApi?.getIsDirty();
    }
}
