import { animate, state, style, transition, trigger } from '@angular/animations';
import {
    ChangeDetectorRef,
    Component,
    EventEmitter,
    Input,
    OnChanges,
    OnDestroy,
    Output,
    SimpleChanges,
    TemplateRef,
    ViewChild
} from '@angular/core';
import { ThemePalette } from '@ptsecurity/mosaic/core';
import { McSidepanelService } from '@ptsecurity/mosaic/sidepanel';
import { Subscription, take } from 'rxjs';

import { ModelsOptionsActions } from '@soldr/api';
import { SharedFacade } from '@soldr/store/shared';

import { LanguageService, LocalizeActionService } from '../../services';
import { EntityModule, EventDetailsItem, EventItem, LocalizedAction, LocalizedField } from '../../types';
import { clone, localizeSchemaAdditionalKeys } from '../../utils';
import { AssigningActionsToEventComponent } from '../assigning-actions-to-event/assigning-actions-to-event.component';
import { NcformWrapperApi } from '../ncform-wrapper/ncform-wrapper.component';

@Component({
    selector: 'soldr-event-details-panel',
    templateUrl: './event-details-panel.component.html',
    styleUrls: ['./event-details-panel.component.scss'],
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
export class EventDetailsPanelComponent implements OnChanges, OnDestroy {
    @Input() assigningActionsToEvent: AssigningActionsToEventComponent;
    @Input() isReadOnly: boolean;
    @Input() module: EntityModule;
    @Input() selectedEvent: EventItem;

    @Output() saveCurrentConfig = new EventEmitter<any>();

    @ViewChild('sidePanel') sidePanel: TemplateRef<any>;

    eventDetails: EventDetailsItem;
    globalActions: ModelsOptionsActions[];
    scrolled = false;
    themePalette = ThemePalette;

    private paramsNcformApi: NcformWrapperApi;
    private panelId = 'event-details-panel';
    private subscription: Subscription = new Subscription();

    constructor(
        private languageService: LanguageService,
        private localizeActionService: LocalizeActionService,
        private sharedFacade: SharedFacade,
        private sidePanelService: McSidepanelService,
        private cdr: ChangeDetectorRef
    ) {
        this.subscription = this.sharedFacade.optionsActions$.subscribe((actions) => (this.globalActions = actions));
    }

    get parametersNumber(): number {
        return Object.keys((this.selectedEvent?.schema.properties as Record<string, any>) || {}).length;
    }

    ngOnChanges({ selectedEvent }: SimpleChanges) {
        if (selectedEvent?.currentValue) {
            const name = this.selectedEvent?.name;
            const event = this.module.current_event_config[name];
            const eventLocale = this.module.locale.events[name];

            const fields = event.fields.map(
                (field: string) =>
                    ({
                        localizedName: this.module.locale.fields[field][this.languageService.lang].title,
                        localizedDescription: this.module.locale.fields[field][this.languageService.lang].description
                    } as LocalizedField)
            );

            this.eventDetails = {
                ...this.selectedEvent,
                fields,
                localizedDescription: eventLocale?.[this.languageService.lang]?.description,
                model: event,
                type: event.type,
                schema: localizeSchemaAdditionalKeys(
                    this.selectedEvent.schema,
                    this.module.locale?.events_additional_args
                )
            } as EventDetailsItem;
        }
    }

    ngOnDestroy() {
        this.subscription.unsubscribe();
    }

    open() {
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
        this.paramsNcformApi.validate().then(({ result }) => {
            if (result) {
                const module = clone(this.module) as EntityModule;
                module.current_event_config[this.selectedEvent.name] = {
                    ...module.current_event_config[this.selectedEvent.name],
                    ...this.paramsNcformApi.getValue(),
                    actions: this.eventDetails.actions
                };
                this.saveCurrentConfig.emit(module);
            }
        });
        this.sidePanelService.getSidepanelById(this.panelId).close();
    }

    cancel() {
        this.sidePanelService.getSidepanelById(this.panelId).close();
    }

    resetToDefault() {
        const name = this.selectedEvent.name;
        const defaultActions = this.module.default_event_config[name]?.actions.map((action) =>
            this.localizeActionService.localizeAction(action, this.globalActions)
        ) as LocalizedAction[];
        const defaultConfig = {
            ...this.module.default_event_config[name],
            actions: defaultActions
        };
        this.eventDetails.actions = [...defaultActions];
        this.eventDetails.model = { ...defaultConfig };
    }

    onScrollBody($event: Event) {
        const headerHeight = 56;
        this.scrolled = ($event.target as HTMLElement).scrollTop > headerHeight;
    }

    onRegisterNcformApi(api: NcformWrapperApi) {
        this.paramsNcformApi = api;
    }

    onModelChange(): void {
        this.cdr.detectChanges();
    }

    get canSave() {
        return this.selectedEvent?.hasParams && this.paramsNcformApi?.getIsDirty();
    }
}
