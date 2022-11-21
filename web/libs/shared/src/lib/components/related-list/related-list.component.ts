import { ChangeDetectionStrategy, Component, Input, TemplateRef, ViewEncapsulation } from '@angular/core';

@Component({
    selector: 'soldr-related-list',
    templateUrl: './related-list.component.html',
    styleUrls: ['./related-list.component.scss'],
    encapsulation: ViewEncapsulation.None,
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class RelatedListComponent {
    @Input() base: any;
    @Input() list: any[];
    @Input() itemTemplate: TemplateRef<any>;
    @Input() popoverItemTemplate: TemplateRef<any>;
}
