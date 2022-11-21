/* eslint-disable @typescript-eslint/ban-ts-comment */
import {
    Component,
    ElementRef,
    EventEmitter,
    forwardRef,
    Input,
    OnChanges,
    OnInit,
    Output,
    SimpleChanges
} from '@angular/core';
import { NG_VALUE_ACCESSOR } from '@angular/forms';
// @ts-ignore
import vueNcform from '@vxcontrol/ncform';
// @ts-ignore
import ncformStdComps from '@vxcontrol/ncform-theme-elementui';
import ElementUI from 'element-ui';
// @ts-ignore
import element_ui_locale_en from 'element-ui/lib/locale/lang/en';
// @ts-ignore
import element_ui_locale_ru from 'element-ui/lib/locale/lang/ru-RU';
// @ts-ignore
import Vue from 'vue';
import VueI18n, { Formatter } from 'vue-i18n';

import { LANGUAGES } from '@soldr/i18n';
import { LanguageService, VueMessageFormatter } from '@soldr/shared';

import { NcformAppComponent } from './ncform-app.component';

export interface NcformWrapperApi {
    validate: () => Promise<any>;
    getValue: () => any;
    getIsDirty: () => boolean;
}

@Component({
    selector: 'soldr-ncform-wrapper',
    templateUrl: './ncform-wrapper.component.html',
    styleUrls: ['./ncform-wrapper.component.scss'],
    providers: [
        {
            provide: NG_VALUE_ACCESSOR,
            useExisting: forwardRef(() => NcformWrapperComponent),
            multi: true
        }
    ]
})
export class NcformWrapperComponent implements OnInit, OnChanges {
    @Input() model: any;
    @Input() schema: any;
    @Input() isReadOnly: boolean;

    @Output() registerApi = new EventEmitter<NcformWrapperApi>();
    @Output() modelChange = new EventEmitter<any>();

    onChange: any;
    onTouched: any;

    private app: Vue;

    constructor(private languageService: LanguageService, private element: ElementRef) {}

    ngOnChanges({ schema, model }: SimpleChanges): void {
        if (!this.app) {
            return;
        }

        if (schema) {
            this.app.$data.schema = this.schema;
        }

        if (model) {
            this.app.$data.model = this.model;
        }
    }

    ngOnInit(): void {
        setTimeout(() => {
            this.initVueApp();
        });
    }

    private initVueApp() {
        const lang = this.languageService.lang;

        Vue.use(VueI18n);
        Vue.use(ElementUI, { locale: this.getElementUiLocaleMessages()[lang] });
        // eslint-disable-next-line @typescript-eslint/no-unsafe-argument
        Vue.use(vueNcform, { extComponents: ncformStdComps, lang });

        const formatter = new VueMessageFormatter({ locale: lang });
        const i18n = new VueI18n({
            locale: lang,
            formatter: formatter as Formatter,
            messages: this.getLocaleMessages()
        });

        const el = this.element.nativeElement.children[0];

        this.app = new Vue({
            el,
            data: {
                model: undefined,
                schema: undefined,
                isReadOnly: false,
                isDirty: false
            },
            methods: {
                onChangeModel: (model: any) => this.modelChange.emit(model)
            },
            i18n,
            // @ts-ignore
            render: (h) => h(NcformAppComponent)
        });

        this.updateInputs();

        this.registerApi.emit(this.getApi());
    }

    private getLocaleMessages() {
        return {
            [LANGUAGES.ru]: {},
            [LANGUAGES.en]: {}
        };
    }

    private getElementUiLocaleMessages() {
        return { [LANGUAGES.en]: element_ui_locale_en, [LANGUAGES.ru]: element_ui_locale_ru };
    }

    private updateInputs() {
        if (this.app) {
            this.app.$data.model = this.model;
            this.app.$data.schema = this.schema;
            this.app.$data.isReadOnly = this.isReadOnly;
        }
    }

    private validate(): Promise<any> {
        // @ts-ignore
        return this.app?.$children[0]?.validate();
    }

    private getValue() {
        // @ts-ignore
        return this.app?.$children[0]?.getValue();
    }

    private getApi() {
        return {
            validate: () => this.validate(),
            getValue: () => this.getValue(),
            getIsDirty: () => this.app.$data.isDirty
        };
    }
}
