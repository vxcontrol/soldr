import { Component, ElementRef, Input, OnDestroy, OnInit, ViewChild } from '@angular/core';
import { FormArray, FormBuilder, Validators } from '@angular/forms';
import { McAutocompleteSelectedEvent } from '@ptsecurity/mosaic/autocomplete';
import { ThemePalette } from '@ptsecurity/mosaic/core';
import { first, Subject, Subscription, take, withLatestFrom } from 'rxjs';

import { DependencyType, ModelsDependencyItem, ModelsModuleS } from '@soldr/api';
import { ModuleEditFacade } from '@soldr/store/modules';

import { ModuleSection } from '../../types';
import { mustBeAgentVersionValidator, mustBeModuleNameValidator, mustBeModuleVersionValidator } from '../../validators';

interface DependencyItem {
    moduleName: string;
    version: string;
}

@Component({
    selector: 'soldr-edit-dependencies-section',
    templateUrl: './edit-dependencies-section.component.html',
    styleUrls: ['./edit-dependencies-section.component.scss']
})
export class EditDependenciesSectionComponent implements OnInit, OnDestroy, ModuleSection {
    @Input() readOnly: boolean;

    @ViewChild('formElement') formElement: ElementRef<HTMLFormElement>;

    agentVersions$ = this.moduleEditFacade.agentVersions$;
    allModules$ = this.moduleEditFacade.allModules$;
    moduleVersionsByName$ = this.moduleEditFacade.moduleVersionsByName$;
    form = this.formBuilder.group({
        agentVersion: this.formBuilder.control('', [mustBeAgentVersionValidator()]),
        receiveData: this.formBuilder.array<DependencyItem>([]),
        sendData: this.formBuilder.array<DependencyItem>([])
    });
    themePalette = ThemePalette;

    private subscription = new Subscription();
    private validationState$ = new Subject<boolean>();

    constructor(private formBuilder: FormBuilder, private moduleEditFacade: ModuleEditFacade) {}

    get receiveDataDependencies(): FormArray {
        return this.form.controls.receiveData;
    }

    get sendDataDependencies(): FormArray {
        return this.form.controls.sendData;
    }

    ngOnInit(): void {
        this.moduleEditFacade.fetchAllModules();
        this.moduleEditFacade.fetchAgentVersions();

        this.watchStore();
        this.watchFormModel();
    }

    ngOnDestroy(): void {
        this.subscription.unsubscribe();
    }

    addReceiveDataDependency(dependency?: ModelsDependencyItem) {
        this.receiveDataDependencies.push(this.getDataTransferFormGroup(dependency));
    }

    addSendDataDependency(dependency?: ModelsDependencyItem) {
        this.sendDataDependencies.push(this.getDataTransferFormGroup(dependency));
    }

    removeReceiveDataDependency(index: number) {
        this.receiveDataDependencies.removeAt(index);
    }

    removeSendDataDependency(index: number) {
        this.sendDataDependencies.removeAt(index);
    }

    loadModuleVersions($event: McAutocompleteSelectedEvent) {
        this.moduleEditFacade.fetchModuleVersionsByName($event.option.value as string);
    }

    onSubmitForm() {
        this.form.statusChanges.pipe(first()).subscribe((schemaStatus) => {
            this.validationState$.next(schemaStatus === 'VALID');
        });

        setTimeout(() => {
            this.form.updateValueAndValidity();
        });
    }

    validateForms() {
        this.formElement.nativeElement.requestSubmit();

        const result$ = this.validationState$.pipe(take(1));

        result$.subscribe((status) => {
            this.moduleEditFacade.setValidationState('dependencies', status);
        });

        return result$;
    }

    filterModuleByName(item: ModelsModuleS, filter = '') {
        return item.info.name.toLowerCase().includes(filter.toLowerCase());
    }

    private watchFormModel() {
        const agentVersionSubscription = this.form.get('agentVersion').valueChanges.subscribe((version: string) => {
            this.moduleEditFacade.changeAgentVersionDependency(version);
        });

        const receiveDataSubscription = this.form
            .get('receiveData')
            .valueChanges.subscribe((dependencies: DependencyItem[]) => {
                const data = dependencies.map((model) => this.getOriginalModel(DependencyType.ToReceiveData, model));
                this.moduleEditFacade.changeReceiveDataDependencies(data);
            });

        const sendDataSubscription = this.form
            .get('sendData')
            .valueChanges.subscribe((dependencies: DependencyItem[]) => {
                const data = dependencies.map((model) => this.getOriginalModel(DependencyType.ToSendData, model));
                this.moduleEditFacade.changeSendDataDependencies(data);
            });

        this.subscription.add(agentVersionSubscription);
        this.subscription.add(receiveDataSubscription);
        this.subscription.add(sendDataSubscription);
    }

    private watchStore() {
        const agentVersionSubscription = this.moduleEditFacade.agentVersionDependency$.subscribe((dependency) => {
            this.form.controls.agentVersion.setValue(dependency?.min_agent_version || '', { emitEvent: false });
        });
        this.subscription.add(agentVersionSubscription);

        const receiveDataDependenciesSubscription = this.moduleEditFacade.receiveDataDependencies$
            .pipe(withLatestFrom(this.moduleEditFacade.moduleVersionsByName$))
            .subscribe(([dependencies, moduleVersionsByName]) => {
                for (let i = 0; i < dependencies.length; i++) {
                    const dependency = dependencies[i];
                    const index = i.toString();

                    if (!this.receiveDataDependencies.get(index)) {
                        this.addReceiveDataDependency(dependency);
                    } else {
                        this.receiveDataDependencies.get(index).setValue(
                            {
                                moduleName: dependency.module_name || '',
                                version: dependency.min_module_version || ''
                            },
                            {
                                emitEvent: false
                            }
                        );
                    }

                    if (!moduleVersionsByName[dependency.module_name]) {
                        this.moduleEditFacade.fetchModuleVersionsByName(dependency.module_name);
                    }
                }
            });
        this.subscription.add(receiveDataDependenciesSubscription);

        const sendDataDependenciesSubscription = this.moduleEditFacade.sendDataDependencies$
            .pipe(withLatestFrom(this.moduleEditFacade.moduleVersionsByName$))
            .subscribe(([dependencies, moduleVersionsByName]) => {
                for (let i = 0; i < dependencies.length; i++) {
                    const dependency = dependencies[i];
                    const index = i.toString();

                    if (!this.sendDataDependencies.get(index)) {
                        this.addSendDataDependency(dependency);
                    } else {
                        this.sendDataDependencies.get(index).setValue(
                            {
                                moduleName: dependency.module_name || '',
                                version: dependency.min_module_version || ''
                            },
                            {
                                emitEvent: false
                            }
                        );
                    }

                    if (!moduleVersionsByName[dependency.module_name]) {
                        this.moduleEditFacade.fetchModuleVersionsByName(dependency.module_name);
                    }
                }
            });
        this.subscription.add(sendDataDependenciesSubscription);

        if (this.readOnly) {
            this.form.disable();
        }
    }

    private getDataTransferFormGroup(dependency?: ModelsDependencyItem) {
        return this.formBuilder.group({
            moduleName: [dependency?.module_name || '', [Validators.required, mustBeModuleNameValidator()]],
            version: [dependency?.min_module_version || '', [mustBeModuleVersionValidator()]]
        });
    }

    private getOriginalModel(type: DependencyType, model: DependencyItem) {
        return {
            type,
            module_name: model.moduleName,
            min_module_version: model.version
        } as ModelsDependencyItem;
    }
}
