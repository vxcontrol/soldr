/* eslint-disable @typescript-eslint/ban-ts-comment */
import { HttpClient } from '@angular/common/http';
import {
    Component,
    ElementRef,
    HostBinding,
    Input,
    OnChanges,
    OnDestroy,
    OnInit,
    SimpleChanges,
    ViewEncapsulation
} from '@angular/core';
import ElementUI from 'element-ui';
// @ts-ignore
import element_ui_locale_en from 'element-ui/lib/locale/lang/en';
// @ts-ignore
import element_ui_locale_ru from 'element-ui/lib/locale/lang/ru-RU';
import * as pb from 'protobufjs';
import { BehaviorSubject, combineLatest, concat, forkJoin, Observable, of, Subject, takeUntil } from 'rxjs';
// @ts-ignore
import Vue from 'vue';
// @ts-ignore
import VueDataTables from 'vue-data-tables';
import VueI18n, { Formatter } from 'vue-i18n';

import { AgentsService, EventsService, GroupsService, PoliciesService } from '@soldr/api';
import { LANGUAGES } from '@soldr/i18n';
import { Entity, EntityModule, LanguageService, ViewMode, VueMessageFormatter } from '@soldr/shared';

// @ts-ignore
import { ModuleEventsApiService, ModulesApiService, NotificationsService } from '../../services';
// @ts-ignore
import { VXAPI } from '../../utils/proto.js';

// @ts-ignore
import { ModuleAppComponent } from './module-app.component.js';

const COMMON_LOCALE_SCOPE = 'common';
const FEATURE_LOCALE_SCOPE = 'modules-interactivity';
const MODULES_LOCALE_SCOPE = 'modules';

type LocaleSet = [VueI18n.LocaleMessageObject, VueI18n.LocaleMessageObject];

@Component({
    selector: 'soldr-module-interactive-part',
    templateUrl: './module-interactive-part.component.html',
    styleUrls: ['./module-interactive-part.component.scss'],
    encapsulation: ViewEncapsulation.None
})
export class ModuleInteractivePartComponent implements OnInit, OnChanges, OnDestroy {
    @Input() viewMode: ViewMode;
    @Input() module: EntityModule;
    @Input() entity: Entity;

    @HostBinding('class.module-interactivity-part') class = true;

    private interactiveModule$ = new BehaviorSubject<any>({});
    private readonly destroyed$: Subject<void> = new Subject();
    private elementUiLocale = {
        [LANGUAGES.en]: element_ui_locale_en as VueI18n.LocaleMessageObject,
        [LANGUAGES.ru]: element_ui_locale_ru as VueI18n.LocaleMessageObject
    };
    private locales$ = concat(
        of([[{}, {}]] as LocaleSet[]),
        forkJoin([
            forkJoin([
                this.httpClient.get(`/assets/i18n/ru-RU/${COMMON_LOCALE_SCOPE}.json`),
                this.httpClient.get(`/assets/i18n/en-US/${COMMON_LOCALE_SCOPE}.json`)
            ]),
            forkJoin([
                this.httpClient.get(`/assets/i18n/ru-RU/${MODULES_LOCALE_SCOPE}.json`),
                this.httpClient.get(`/assets/i18n/en-US/${MODULES_LOCALE_SCOPE}.json`)
            ]),
            forkJoin([
                this.httpClient.get(`/assets/i18n/ru-RU/${FEATURE_LOCALE_SCOPE}.json`),
                this.httpClient.get(`/assets/i18n/en-US/${FEATURE_LOCALE_SCOPE}.json`)
            ])
        ] as Observable<LocaleSet>[])
    );
    private app: Vue;

    constructor(
        private httpClient: HttpClient,
        private languageService: LanguageService,
        private element: ElementRef,
        private agentsService: AgentsService,
        private eventsService: EventsService,
        private groupsService: GroupsService,
        private policiesService: PoliciesService
    ) {}

    ngOnInit(): void {
        Promise.all([pb.load('/assets/proto/agent.proto'), pb.load('/assets/proto/protocol.proto')]).then(
            ([agentProto, protocolProto]: [any, any]) => {
                this.initVueApp(agentProto, protocolProto);
            }
        );

        combineLatest([this.locales$, this.interactiveModule$])
            .pipe(takeUntil(this.destroyed$))
            .subscribe(([appLocalization, module]) => {
                const moduleLocalization = this.getModuleLocalization(module);
                const interactiveLocalization = [...appLocalization, ...moduleLocalization];
                this.setAppLocale(interactiveLocalization);
            });
    }

    ngOnChanges({ module }: SimpleChanges) {
        if (module?.currentValue && this.viewMode !== ViewMode.Policies) {
            this.interactiveModule$.next(this.module);
        }
    }

    ngOnDestroy() {
        this.destroyed$.next();
        this.destroyed$.complete();
    }

    private getModuleLocalization(module: any): LocaleSet[] {
        const ui = module?.locale?.ui as Record<string, Record<string, string>>;

        return [
            Object.keys(ui || {}).reduce(
                (array, key) => {
                    Object.keys(ui[key]).forEach(
                        (lang, index) => (array[index] = { ...array[index], [key]: ui[key][lang] })
                    );

                    return array;
                },
                [{}, {}]
            )
        ];
    }

    private initVueApp(agentProto: any, protocolProto: any) {
        const lang = this.languageService.lang;

        Vue.use(VueI18n);
        Vue.use(ElementUI, { locale: this.elementUiLocale[lang] });
        // eslint-disable-next-line @typescript-eslint/no-unsafe-argument
        Vue.use(VueDataTables);

        const formatter = new VueMessageFormatter({ locale: lang });
        const i18n = new VueI18n({
            locale: lang,
            formatter: formatter as Formatter
        });
        const modulesAPI = new ModulesApiService(
            {
                id: this.entity.id,
                hash: this.entity.hash,
                module_name: this.module.info.name,
                viewMode: this.viewMode
            },
            this.agentsService,
            this.groupsService,
            this.policiesService
        );
        const eventsAPI = new ModuleEventsApiService(
            {
                id: this.entity.id,
                hash: this.entity.hash,
                module_name: this.module.info.name,
                viewMode: this.viewMode
            },
            this.agentsService,
            this.eventsService,
            this.groupsService,
            this.policiesService
        );
        const subview = modulesAPI.getView(this.module.info.name);
        let protoAPI: VXAPI;
        let vxHostPort: string;
        if (window.location.protocol === 'https:') {
            vxHostPort = `wss://${window.location.host}`;
        } else {
            vxHostPort = `ws://${window.location.host}`;
        }

        if (this.viewMode === ViewMode.Agents) {
            protoAPI = new VXAPI({
                hash: this.entity.hash,
                type: "browser",
                moduleName: this.module.info.name,
                hostPort: vxHostPort,
                agentProto,
                protocolProto
            });
        } else if (this.viewMode === ViewMode.Groups) {
            protoAPI = new VXAPI({
                hash: this.entity.hash,
                type: "aggregate",
                moduleName: this.module.info.name,
                hostPort: vxHostPort,
                agentProto,
                protocolProto
            });
        }

        this.app = new Vue({
            el: (this.element.nativeElement as Element).querySelector('.app'),
            data: {
                viewMode: this.viewMode.replace(/s$/, ''),
                module: this.module,
                entity: this.entity,
                api: {
                    modulesAPI,
                    eventsAPI
                },
                protoAPI,
                subview
            },
            methods: {},
            i18n,
            // @ts-ignore
            render: (h) => h(ModuleAppComponent)
        });

        this.app.$root.NotificationsService = new NotificationsService(this.app);
    }

    private setAppLocale(locales: LocaleSet[]) {
        this.app?.$i18n.setLocaleMessage('ru', { ...locales.reduce((acc, item) => ({ ...acc, ...item[0] }), {}) });
        this.app?.$i18n.setLocaleMessage('en', { ...locales.reduce((acc, item) => ({ ...acc, ...item[1] }), {}) });
    }
}
