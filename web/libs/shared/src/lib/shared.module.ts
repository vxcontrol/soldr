import { A11yModule } from '@angular/cdk/a11y';
import { CommonModule } from '@angular/common';
import { NgModule } from '@angular/core';
import { FormsModule, ReactiveFormsModule } from '@angular/forms';
import { RouterModule } from '@angular/router';
import { TranslocoModule } from '@ngneat/transloco';
import { LetModule } from '@ngrx/component';
import { DateAdapter, MC_DATE_LOCALE } from '@ptsecurity/cdk/datetime';
import { LuxonDateAdapter, McLuxonDateModule } from '@ptsecurity/mosaic-luxon-adapter/adapter';
import { DateFormatter } from '@ptsecurity/mosaic/core';
import { MC_TOAST_CONFIG, McToastPosition } from '@ptsecurity/mosaic/toast';
import { AgGridModule } from 'ag-grid-angular';

import {
    AccordionComponent,
    ActionDetailsPanelComponent,
    ActionInfoComponent,
    ActionPriorityComponent,
    ActionbarComponent,
    AgentAuthStatusComponent,
    AgentConnectionStatusComponent,
    AgentInfoComponent,
    AgentVersionComponent,
    AgentsGridComponent,
    AssigningActionsMasterComponent,
    AssigningActionsToEventComponent,
    AutocompleteTagsComponent,
    BreadcrumbsComponent,
    ChangelogComponent,
    ColumnComponent,
    ConsistencyIconComponent,
    CopyComponent,
    DeleteModuleFromPolicyModalComponent,
    DependenciesGridComponent,
    DependenciesInfoComponent,
    DependencyStatusComponent,
    DividerComponent,
    SoldrUnavailableActionForEventDirective,
    EventDataBlockComponent,
    EventDetailsPanelComponent,
    EventInfoComponent,
    EventsGridComponent,
    FilterComponent,
    FilterItemComponent,
    GridActionBarDirective,
    GridComponent,
    GridFooterDirective,
    GroupInfoComponent,
    GroupLinkComponent,
    GroupsAndFiltersComponent,
    GroupsGridComponent,
    LinkGroupToPolicyComponent,
    ModuleActionsComponent,
    ModuleConfigBlockComponent,
    ModuleConfigComponent,
    ModuleEventsComponent,
    ModuleInfoComponent,
    ModuleTypeComponent,
    ModulesConfigComponent,
    MultipleSelectionPanelComponent,
    NcformWrapperComponent,
    NoRowsOverlayComponent,
    OsComponent,
    PoliciesGridComponent,
    PolicyInfoComponent,
    ProgressContainerComponent,
    ProgressSpinnerDirective,
    RelatedListComponent,
    SecureModuleConfigComponent,
    TagComponent,
    TemplateCellComponent,
    TextOverflowComponent,
    TextOverflowDirective,
    TreePopoverComponent,
    UpgradeStatusMessageComponent,
    EmptyGridContentComponent
} from './components';
import {
    SoldrGridScrollToBodyEndDirective,
    SaveStateDirective,
    STATE_STORAGE_TOKEN,
    TemplateColumnDirective,
    WidthChangeDirective,
    AutofocusDirective
} from './directives';
import { MosaicModule } from './mosaic.module';
import {
    AbsoluteLongDatePipe,
    AbsoluteLongDateTimePipe,
    AbsoluteLongDateTimeWithSecondsPipe,
    AbsoluteShortDateTimePipe,
    AgentVersionPipe,
    ConvertBytesPipe,
    DaysBeforePipe,
    FilterPipe,
    HashPipe,
    KeysPipe,
    LastConnectedPipe,
    LocalizedDateToDateTimePipe,
    ModuleVersionPipe,
    RelativeLongDateTimePipe,
    RelativeShortDateTimePipe,
    SearchPipe,
    SortPipe,
    ToDateTimePipe
} from './pipes';
import { HbPipe } from './pipes/hb.pipe';
import { ExporterService, ModalInfoService, PageTitleService, StateStorageService } from './services';

const directives = [
    AutofocusDirective,
    SoldrGridScrollToBodyEndDirective,
    SoldrUnavailableActionForEventDirective,
    GridActionBarDirective,
    GridFooterDirective,
    ProgressSpinnerDirective,
    SaveStateDirective,
    TemplateColumnDirective,
    TextOverflowDirective,
    WidthChangeDirective
];

const components = [
    AccordionComponent,
    ActionDetailsPanelComponent,
    ActionInfoComponent,
    ActionPriorityComponent,
    ActionbarComponent,
    AgentAuthStatusComponent,
    AgentConnectionStatusComponent,
    AgentInfoComponent,
    AgentVersionComponent,
    AgentsGridComponent,
    AssigningActionsMasterComponent,
    AssigningActionsToEventComponent,
    AutocompleteTagsComponent,
    BreadcrumbsComponent,
    ChangelogComponent,
    ColumnComponent,
    ConsistencyIconComponent,
    CopyComponent,
    DeleteModuleFromPolicyModalComponent,
    DependenciesGridComponent,
    DependenciesInfoComponent,
    DependencyStatusComponent,
    DividerComponent,
    EmptyGridContentComponent,
    EventDataBlockComponent,
    EventDetailsPanelComponent,
    EventDetailsPanelComponent,
    EventInfoComponent,
    EventsGridComponent,
    FilterComponent,
    FilterItemComponent,
    GridComponent,
    GroupInfoComponent,
    GroupLinkComponent,
    GroupsAndFiltersComponent,
    GroupsGridComponent,
    LinkGroupToPolicyComponent,
    ModuleActionsComponent,
    ModuleConfigBlockComponent,
    ModuleConfigComponent,
    ModuleEventsComponent,
    ModuleInfoComponent,
    ModuleTypeComponent,
    ModulesConfigComponent,
    MultipleSelectionPanelComponent,
    NcformWrapperComponent,
    NoRowsOverlayComponent,
    OsComponent,
    PoliciesGridComponent,
    PolicyInfoComponent,
    ProgressContainerComponent,
    ProgressSpinnerDirective,
    RelatedListComponent,
    SecureModuleConfigComponent,
    TagComponent,
    TemplateCellComponent,
    TextOverflowComponent,
    TextOverflowDirective,
    TreePopoverComponent,
    TreePopoverComponent,
    UpgradeStatusMessageComponent
];

const pipes = [
    AbsoluteLongDatePipe,
    AbsoluteLongDateTimePipe,
    AbsoluteLongDateTimeWithSecondsPipe,
    AbsoluteShortDateTimePipe,
    AgentVersionPipe,
    ConvertBytesPipe,
    DaysBeforePipe,
    FilterPipe,
    HashPipe,
    HbPipe,
    KeysPipe,
    LastConnectedPipe,
    LocalizedDateToDateTimePipe,
    ModuleVersionPipe,
    RelativeLongDateTimePipe,
    RelativeShortDateTimePipe,
    SearchPipe,
    SortPipe,
    ToDateTimePipe
];

@NgModule({
    imports: [
        A11yModule,
        AgGridModule.withComponents([]),
        CommonModule,
        FormsModule,
        McLuxonDateModule,
        MosaicModule,
        ReactiveFormsModule,
        RouterModule,
        TranslocoModule,
        LetModule
    ],
    exports: [...components, ...directives, ...pipes, MosaicModule, TranslocoModule],
    providers: [
        AbsoluteLongDateTimeWithSecondsPipe,
        ConvertBytesPipe,
        ExporterService,
        ModalInfoService,
        PageTitleService,
        RelativeLongDateTimePipe,
        ToDateTimePipe,
        AgentVersionPipe,
        { provide: DateAdapter, useClass: LuxonDateAdapter, deps: [MC_DATE_LOCALE] },
        { provide: DateFormatter, deps: [DateAdapter, MC_DATE_LOCALE] },
        { provide: STATE_STORAGE_TOKEN, useClass: StateStorageService },
        {
            provide: MC_TOAST_CONFIG,
            useValue: {
                position: McToastPosition.TOP_RIGHT,
                duration: 5000,
                delay: 2000,
                onTop: true
            }
        }
    ],
    declarations: [...components, ...directives, ...pipes]
})
export class SharedModule {}
