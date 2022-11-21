import { NgModule } from '@angular/core';
import { McAutocompleteModule } from '@ptsecurity/mosaic/autocomplete';
import { McButtonModule } from '@ptsecurity/mosaic/button';
import { McButtonToggleModule } from '@ptsecurity/mosaic/button-toggle';
import { McCardModule } from '@ptsecurity/mosaic/card';
import { McCheckboxModule } from '@ptsecurity/mosaic/checkbox';
import { McDatepickerModule } from '@ptsecurity/mosaic/datepicker';
import { McDividerModule } from '@ptsecurity/mosaic/divider';
import { McDlModule } from '@ptsecurity/mosaic/dl';
import { McDropdownModule } from '@ptsecurity/mosaic/dropdown';
import { McFormFieldModule } from '@ptsecurity/mosaic/form-field';
import { McIconModule } from '@ptsecurity/mosaic/icon';
import { McInputModule } from '@ptsecurity/mosaic/input';
import { McLinkModule } from '@ptsecurity/mosaic/link';
import { McListModule } from '@ptsecurity/mosaic/list';
import { McLoaderOverlayModule } from '@ptsecurity/mosaic/loader-overlay';
import { McModalModule } from '@ptsecurity/mosaic/modal';
import { McNavbarModule } from '@ptsecurity/mosaic/navbar';
import { McPopoverModule } from '@ptsecurity/mosaic/popover';
import { McProgressBarModule } from '@ptsecurity/mosaic/progress-bar';
import { McProgressSpinnerModule } from '@ptsecurity/mosaic/progress-spinner';
import { McRadioModule } from '@ptsecurity/mosaic/radio';
import { McSelectModule } from '@ptsecurity/mosaic/select';
import { McSidebarModule } from '@ptsecurity/mosaic/sidebar';
import { McSidepanelModule } from '@ptsecurity/mosaic/sidepanel';
import { McSplitterModule } from '@ptsecurity/mosaic/splitter';
import { McTableModule } from '@ptsecurity/mosaic/table';
import { McTabsModule } from '@ptsecurity/mosaic/tabs';
import { McTagsModule } from '@ptsecurity/mosaic/tags';
import { McTextareaModule } from '@ptsecurity/mosaic/textarea';
import { McTimepickerModule } from '@ptsecurity/mosaic/timepicker';
import { McToastModule, McToastService } from '@ptsecurity/mosaic/toast';
import { McToolTipModule } from '@ptsecurity/mosaic/tooltip';
import { McTreeModule } from '@ptsecurity/mosaic/tree';
import { McHighlightModule } from '@ptsecurity/mosaic/core';

const mosaicModules = [
    McAutocompleteModule,
    McButtonModule,
    McButtonToggleModule,
    McCardModule,
    McCheckboxModule,
    McDatepickerModule,
    McDividerModule,
    McDlModule,
    McDropdownModule,
    McFormFieldModule,
    McHighlightModule,
    McIconModule,
    McInputModule,
    McLinkModule,
    McListModule,
    McLoaderOverlayModule,
    McModalModule,
    McNavbarModule,
    McPopoverModule,
    McProgressBarModule,
    McProgressSpinnerModule,
    McRadioModule,
    McSelectModule,
    McSidebarModule,
    McSidepanelModule,
    McSplitterModule,
    McTableModule,
    McTabsModule,
    McTagsModule,
    McTextareaModule,
    McTimepickerModule,
    McToastModule,
    McToolTipModule,
    McTreeModule
];

@NgModule({
    declarations: [],
    imports: [...mosaicModules],
    exports: [...mosaicModules],
    providers: [McToastService]
})
export class MosaicModule {}
