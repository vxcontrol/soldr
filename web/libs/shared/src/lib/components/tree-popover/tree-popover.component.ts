import {
    Component,
    EventEmitter,
    Input,
    OnChanges,
    OnDestroy,
    OnInit,
    Output,
    SimpleChanges,
    ViewChild,
    ViewEncapsulation
} from '@angular/core';
import { ThemePalette } from '@ptsecurity/mosaic/core';
import { McInput } from '@ptsecurity/mosaic/input';
import { McPopoverComponent } from '@ptsecurity/mosaic/popover';
import { ReplaySubject, Subscription } from 'rxjs';

import { ListItem } from '../../types';

@Component({
    selector: 'soldr-tree-popover',
    templateUrl: './tree-popover.component.html',
    styleUrls: ['./tree-popover.component.scss'],
    encapsulation: ViewEncapsulation.None
})
export class TreePopoverComponent implements OnInit, OnChanges, OnDestroy {
    @Input() headerText: string;
    @Input() resetText: string;
    @Input() searchPlaceholder: string;
    @Input() items: ListItem[] = [];
    @Input() selected: string[];

    @Output() search = new EventEmitter<string>();
    @Output() apply = new EventEmitter<string[]>();

    @ViewChild('popover', { static: false }) popover: McPopoverComponent;
    @ViewChild('searchInput') searchInput: McInput;

    foundItems$ = new ReplaySubject<ListItem[]>(1);
    themePalette = ThemePalette;
    searchValue$ = new ReplaySubject<string>(1);
    searchValue: string;
    selected$ = new ReplaySubject<string[]>(1);
    selectedItems: string[] = [];
    initialSelectedItems: string[] = [];
    subscription = new Subscription();

    constructor() {}

    ngOnInit(): void {
        this.searchValue$.next('');

        const searchSubscription = this.searchValue$.subscribe((value) => {
            this.search.emit(value);
        });
        const selectionSubscription = this.selected$.subscribe((value) => {
            this.selectedItems = value;
            this.initialSelectedItems = value;
        });

        this.subscription.add(selectionSubscription);
        this.subscription.add(searchSubscription);
    }

    ngOnChanges({ items, selected }: SimpleChanges): void {
        if (items?.currentValue) {
            this.foundItems$.next(this.items);
        }

        if (selected?.currentValue) {
            this.selected$.next(this.selected);
        }
    }

    ngOnDestroy(): void {
        this.subscription.unsubscribe();
    }

    onReset() {
        this.selectedItems = [];
        this.apply.emit(this.selectedItems);
        this.popover.hide(0);
    }

    onApply() {
        this.apply.emit(this.selectedItems);
        this.search.emit('');
        this.popover.hide(0);
    }

    onCancel() {
        this.selectedItems = this.initialSelectedItems;
        this.popover.hide(0);
    }

    onPopoverVisibleChange(value: boolean) {
        if (!value) {
            this.searchValue = '';
            this.searchValue$.next('');
        }
    }
}
