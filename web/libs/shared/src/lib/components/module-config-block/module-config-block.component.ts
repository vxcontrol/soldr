import {
    Component,
    EventEmitter,
    Input,
    OnChanges,
    OnDestroy,
    OnInit,
    Output,
    SimpleChanges,
    ViewChild
} from '@angular/core';
import { Router } from '@angular/router';
import { BehaviorSubject, combineLatest, filter, map, Observable, ReplaySubject, Subject, takeUntil } from 'rxjs';

import { LanguageService } from '../../services';
import { EntityModule, ReadOnlyModule, ViewMode } from '../../types';
import { compareObjects } from '../../utils';
import { ModuleConfigComponent } from '../module-config/module-config.component';

@Component({
    selector: 'soldr-module-config-block',
    templateUrl: './module-config-block.component.html',
    styleUrls: ['./module-config-block.component.scss']
})
export class ModuleConfigBlockComponent implements OnInit, OnChanges, OnDestroy {
    @Input() isReadOnly: boolean;
    @Input() module: EntityModule;
    @Input() policyHash: string;
    @Input() viewMode: ViewMode;

    @Output() saveModuleConfig = new EventEmitter();

    @ViewChild(ModuleConfigComponent) moduleConfig: ModuleConfigComponent;

    canShowActions$: Observable<boolean>;
    canShowConfig$: Observable<boolean>;
    isReadOnly$ = new BehaviorSubject<boolean>(true);
    module$ = new BehaviorSubject<EntityModule>(undefined);
    moduleConfig$ = new ReplaySubject(1);
    policyLink$: Observable<string>;

    canSaveModuleConfig = false;
    viewModeEnum = ViewMode;

    private readonly destroyed$: Subject<void> = new Subject();

    constructor(private languageService: LanguageService, private router: Router) {}

    ngOnInit(): void {
        this.canShowConfig$ = this.module$.pipe(
            filter(Boolean),
            map(
                ({ config_schema, secure_current_config }) =>
                    Object.keys(config_schema.properties as Record<string, any>).length > 0 ||
                    (Object.keys((secure_current_config as object) || {}).length > 0 &&
                        this.viewMode === ViewMode.Policies)
            )
        );
        this.canShowActions$ = this.module$.pipe(
            filter(Boolean),
            map(
                ({ action_config_schema }) =>
                    Object.keys(action_config_schema.properties as Record<string, any>).length > 0
            )
        );
        this.policyLink$ = combineLatest([this.module$, this.isReadOnly$]).pipe(
            filter(([module, isReadOnly]) => isReadOnly && !!(module as ReadOnlyModule)?.details?.policy),
            map(([module]) => {
                const path = this.router
                    .createUrlTree(['/policies', (module as ReadOnlyModule).details.policy.hash], {
                        queryParams: { tab: 'modules', moduleName: module.info.name }
                    })
                    .toString();

                return `<a class="mc-link" href="${path}">
                            ${(module as ReadOnlyModule).details.policy.info.name[this.languageService.lang]}
                        </a>`;
            })
        );

        combineLatest([this.moduleConfig$, this.module$])
            .pipe(takeUntil(this.destroyed$))
            .subscribe(
                ([currentConfig, module]) =>
                    (this.canSaveModuleConfig = !compareObjects(currentConfig, module.current_config))
            );
    }

    save(module: EntityModule) {
        this.saveModuleConfig.emit(module);
    }

    validate() {
        return this.moduleConfig?.validate();
    }

    reset() {
        return this.moduleConfig?.reset();
    }

    getModel() {
        return this.moduleConfig?.getModel();
    }

    changeConfig(config: any) {
        this.moduleConfig$.next(config);
    }

    ngOnChanges({ module, isReadOnly }: SimpleChanges): void {
        if (module?.currentValue) {
            this.module$.next(this.module);
        }

        if (isReadOnly) {
            this.isReadOnly$.next(this.isReadOnly);
        }
    }

    ngOnDestroy() {
        this.destroyed$.next();
        this.destroyed$.complete();
    }

    get isDirty() {
        return this.moduleConfig?.isDirty;
    }
}
