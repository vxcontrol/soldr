import { By } from '@angular/platform-browser';
import { AVAILABLE_LOCALES, DEFAULT_LOCALE, LOCALES } from '@soldr/i18n';
import { EdrGridScrollToBodyEndDirective, SharedModule, SortingDirection, TemplateCellComponent } from '@soldr/shared';
import { TranslocoTestingModule } from '@ngneat/transloco';
import { AgGridModule } from 'ag-grid-angular';
import { MockBuilder, MockRender, ngMocks } from 'ng-mocks';

import { AVAILABLE_LOCALES, DEFAULT_LOCALE, LOCALES } from '@soldr/i18n';
import { SoldrGridScrollToBodyEndDirective, SharedModule, SortingDirection, TemplateCellComponent } from '@soldr/shared';

import { MosaicModule } from '../../mosaic.module';

import { GridComponent } from './grid.component';

const data = [
    { id: 1, value: 'value1' },
    { id: 2, value: 'value2' },
    { id: 3, value: 'value3' }
];
Object.freeze(data);

describe('GridComponent', () => {
    beforeEach(() =>
        MockBuilder(GridComponent, SharedModule)
            .keep(MosaicModule)
            .keep(TemplateCellComponent)
            .keep(SoldrGridScrollToBodyEndDirective)
            .keep(AgGridModule.withComponents([]))
            .keep(
                TranslocoTestingModule.forRoot({
                    langs: {
                        [LOCALES.ru_RU]: {},
                        [LOCALES.en_US]: {}
                    },
                    preloadLangs: true,
                    translocoConfig: {
                        availableLangs: AVAILABLE_LOCALES,
                        defaultLang: DEFAULT_LOCALE
                    }
                }),
              {
                export: true
              }
            )
    );

    it('should create', () => {
        const fixture = MockRender<GridComponent>(GridComponent);
        const component = fixture.point.componentInstance;
        expect(component).toBeTruthy();
    });

    describe('Grid', () => {
        it('should render headers', async () => {
            const fixture = MockRender<GridComponent>(
                `
                    <soldr-grid [data]="data">
                        <soldr-column field="id" [headerName]="'ID'" [default]="true"></soldr-column>
                        <soldr-column field="value" [headerName]="'Value'" [default]="true"></soldr-column>
                    </soldr-grid>
                `,
                {
                    data
                }
            );
            await fixture.whenStable();
            const headers = fixture.debugElement.queryAll(By.css('.ag-header-cell-text'));

            expect(headers.length).toBe(2);
            expect(headers[0].nativeElement.textContent).toBe('ID');
            expect(headers[1].nativeElement.textContent).toBe('Value');
        });

        it('should render simple cells', async () => {
            const fixture = MockRender<GridComponent>(
                `
                    <soldr-grid [data]="data">
                        <soldr-column field="id" [headerName]="'ID'"></soldr-column>
                        <soldr-column field="value" [headerName]="'Value'"></soldr-column>
                    </soldr-grid>
                `,
                {
                    data
                }
            );
            await fixture.whenStable();

            expect(fixture.debugElement.queryAll(By.css('.ag-cell')).length).toBe(data.length * 2);
        });

        it('should render customized cells', async () => {
            const fixture = MockRender<GridComponent>(
                `
                    <soldr-grid [data]="data">
                        <ng-template #valueColumnTemplate let-node="params.data">
                            <span class="custom-cell">{{ node.value }}}</span>
                        </ng-template>
                        <soldr-column field="id" [headerName]="'ID'"></soldr-column>
                        <soldr-column field="value" [headerName]="'Value'" [template]="valueColumnTemplate"></soldr-column>
                    </soldr-grid>
                `,
                {
                    data
                }
            );
            await fixture.whenStable();

            expect(fixture.debugElement.queryAll(By.css('.custom-cell')).length).toBe(data.length);
            expect(
                fixture.debugElement.queryAll(By.css('.custom-cell'))[0].nativeElement.textContent.includes('value1')
            ).toBeTruthy();
        });

        it('should render footer', async () => {
            const fixture = MockRender<GridComponent>(
                `
                    <soldr-grid [data]="data" [total]="total">
                        <soldr-column field="id" [headerName]="'ID'"></soldr-column>
                        <soldr-column field="value" [headerName]="'Value'"></soldr-column>
                    </soldr-grid>
                `,
                {
                    data,
                    total: data.length
                }
            );
            await fixture.whenStable();

            expect(fixture.debugElement.queryAll(By.css('.grid__footer-items span')).length).toBe(1);
        });
    });

    describe('Selection', () => {
        it('should work in single mode', async () => {
            const fixture = MockRender<GridComponent>(
                `
                    <soldr-grid [data]="data" [total]="total" (selectRows)="onSelect($event)">
                        <soldr-column field="id" [headerName]="'ID'"></soldr-column>
                        <soldr-column field="value" [headerName]="'Value'"></soldr-column>
                    </soldr-grid>
                `,
                {
                    data,
                    total: data.length
                }
            );
            await fixture.whenStable();

            const component = fixture.componentInstance;
            ngMocks.stubMember(component, 'onSelect', jest.fn());
            fixture.debugElement.queryAll(By.css(':not(.ag-hidden)>.ag-row'))[0].nativeElement.click();
            await fixture.whenStable();

            expect(fixture.debugElement.queryAll(By.css(':not(.ag-hidden)>.ag-row-selected')).length).toBe(1);
            expect(fixture.debugElement.queryAll(By.css('.grid__footer-items span')).length).toBe(1);
            expect(component.onSelect).toHaveBeenCalledWith([data[0]]);

            fixture.debugElement.queryAll(By.css(':not(.ag-hidden)>.ag-row'))[1].nativeElement.click();
            await fixture.whenStable();

            expect(component.onSelect).toHaveBeenCalledWith([data[1]]);

            fixture.debugElement.queryAll(By.css(':not(.ag-hidden)>.ag-row'))[2].nativeElement.click();
            await fixture.whenStable();

            expect(fixture.debugElement.queryAll(By.css(':not(.ag-hidden)>.ag-row-selected')).length).toBe(1);
            expect(component.onSelect).toHaveBeenCalledWith([data[2]]);
        });

        it('should select only one row in single mode on pressed ctrl', async () => {
            const fixture = MockRender<GridComponent>(
                `
                    <soldr-grid [data]="data" [total]="total" (selectRows)="onSelect($event)">
                        <soldr-column field="id" [headerName]="'ID'"></soldr-column>
                        <soldr-column field="value" [headerName]="'Value'"></soldr-column>
                    </soldr-grid>
                `,
                {
                    data,
                    total: data.length
                }
            );
            const component = fixture.componentInstance;
            ngMocks.stubMember(component, 'onSelect', jest.fn());
            await fixture.whenStable();

            const rows = fixture.debugElement.queryAll(By.css(':not(.ag-hidden)>.ag-row'));
            rows[0].nativeElement.click();
            await fixture.whenStable();
            const event = new MouseEvent('click', {
                view: window,
                bubbles: true,
                ctrlKey: true
            });
            rows[2].nativeElement.dispatchEvent(event);
            await fixture.whenStable();

            expect(fixture.debugElement.queryAll(By.css(':not(.ag-hidden)>.ag-row-selected')).length).toBe(1);
            expect(component.onSelect).toHaveBeenCalledWith([data[2]]);
        });

        it('should select only one row in single mode on pressed shift', async () => {
            const fixture = MockRender<GridComponent>(
                `
                    <soldr-grid [data]="data" [total]="total" (selectRows)="onSelect($event)">
                        <soldr-column field="id" [headerName]="'ID'"></soldr-column>
                        <soldr-column field="value" [headerName]="'Value'"></soldr-column>
                    </soldr-grid>
                `,
                {
                    data,
                    total: data.length
                }
            );
            const component = fixture.componentInstance;
            ngMocks.stubMember(component, 'onSelect', jest.fn());
            await fixture.whenStable();

            const rows = fixture.debugElement.queryAll(By.css(':not(.ag-hidden)>.ag-row'));
            rows[0].nativeElement.click();
            await fixture.whenStable();
            const event = new MouseEvent('click', {
                view: window,
                bubbles: true,
                shiftKey: true
            });
            rows[2].nativeElement.dispatchEvent(event);
            await fixture.whenStable();

            expect(fixture.debugElement.queryAll(By.css(':not(.ag-hidden)>.ag-row-selected')).length).toBe(1);
            expect(component.onSelect).toHaveBeenCalledWith([data[2]]);
        });

        it('should work in multiple mode', async () => {
            const fixture = MockRender<GridComponent>(
                `
                    <soldr-grid [data]="data" [total]="total" (selectRows)="onSelect($event)" [selectionType]="'multiple'">
                        <soldr-column field="id" [headerName]="'ID'"></soldr-column>
                        <soldr-column field="value" [headerName]="'Value'"></soldr-column>
                    </soldr-grid>
                `,
                {
                    data,
                    total: data.length
                }
            );
            const component = fixture.componentInstance;
            ngMocks.stubMember(component, 'onSelect', jest.fn());
            await fixture.whenStable();

            fixture.debugElement.queryAll(By.css(':not(.ag-hidden)>.ag-row'))[0].nativeElement.click();
            await fixture.whenStable();

            expect(fixture.debugElement.queryAll(By.css(':not(.ag-hidden)>.ag-row-selected')).length).toBe(1);
            expect(fixture.debugElement.queryAll(By.css('.grid__footer-items span')).length).toBe(2);
            expect(component.onSelect).toHaveBeenCalledWith([data[0]]);
        });

        it('should work in multiple mode on pressed ctrl', async () => {
            const fixture = MockRender<GridComponent>(
                `
                    <soldr-grid [data]="data" [total]="total" (selectRows)="onSelect($event)" [selectionType]="'multiple'">
                        <soldr-column field="id" [headerName]="'ID'"></soldr-column>
                        <soldr-column field="value" [headerName]="'Value'"></soldr-column>
                    </soldr-grid>
                `,
                {
                    data,
                    total: data.length
                }
            );
            const component = fixture.componentInstance;
            ngMocks.stubMember(component, 'onSelect', jest.fn());
            await fixture.whenStable();

            fixture.debugElement.queryAll(By.css(':not(.ag-hidden)>.ag-row'))[0].nativeElement.click();
            await fixture.whenStable();
            const event = new MouseEvent('click', {
                view: window,
                bubbles: true,
                ctrlKey: true
            });
            fixture.debugElement.queryAll(By.css(':not(.ag-hidden)>.ag-row'))[2].nativeElement.dispatchEvent(event);
            await fixture.whenStable();

            expect(fixture.debugElement.queryAll(By.css(':not(.ag-hidden)>.ag-row-selected')).length).toBe(2);
            expect(component.onSelect).toHaveBeenCalledWith([data[0], data[2]]);
        });

        it('should work in multiple mode on pressed shift', async () => {
            const fixture = MockRender<GridComponent>(
                `
                    <soldr-grid [data]="data" [total]="total" (selectRows)="onSelect($event)" [selectionType]="'multiple'">
                        <soldr-column field="id" [headerName]="'ID'"></soldr-column>
                        <soldr-column field="value" [headerName]="'Value'"></soldr-column>
                    </soldr-grid>
                `,
                {
                    data,
                    total: data.length
                }
            );
            const component = fixture.componentInstance;
            ngMocks.stubMember(component, 'onSelect', jest.fn());
            await fixture.whenStable();

            fixture.debugElement.queryAll(By.css(':not(.ag-hidden)>.ag-row'))[0].nativeElement.click();
            await fixture.whenStable();
            const event = new MouseEvent('click', {
                view: window,
                bubbles: true,
                shiftKey: true
            });
            fixture.debugElement.queryAll(By.css(':not(.ag-hidden)>.ag-row'))[2].nativeElement.dispatchEvent(event);
            await fixture.whenStable();

            const selectedAfterSelectionWithShift = 3;
            expect(fixture.debugElement.queryAll(By.css(':not(.ag-hidden)>.ag-row-selected')).length).toBe(
                selectedAfterSelectionWithShift
            );
            expect(component.onSelect).toHaveBeenCalledWith([data[0], data[1], data[2]]);
        });
    });

    describe('Filtration', () => {
        it('should be rendered if define filter params', () => {
            const fixture = MockRender<GridComponent>(
                `
                    <soldr-grid [data]="data">
                        <soldr-column field="id" [headerName]="'ID'"></soldr-column>
                        <soldr-column field="value" [headerName]="'Value'"></soldr-column>
                        <soldr-filter
                            [field]="'id'"
                            [title]="'Filter By Id'"
                            [multiple]="true">
                            <soldr-filter-item *ngFor="let item of filterItems" [label]="item.label" [value]="item.value">
                            </soldr-filter-item>
                        </soldr-filter>
                    </soldr-grid>
                `,
                {
                    data,
                    filterItems: data.map((item) => ({ label: item.id, value: item.id }))
                }
            );

            expect(fixture.debugElement.queryAll(By.css('.grid__button_show-filters')).length).toBe(1);
            expect(fixture.debugElement.queryAll(By.css('.grid__button_show-filters .mc-badge')).length).toBe(0);
        });

        it('should open block with filters and reset button when click on "Show Filters" button', () => {
            const fixture = MockRender<GridComponent>(
                `
                    <soldr-grid [data]="data">
                        <soldr-column field="id" [headerName]="'ID'"></soldr-column>
                        <soldr-column field="value" [headerName]="'Value'"></soldr-column>
                        <soldr-filter
                            [field]="'id'"
                            [title]="'Filter By Id'"
                            [multiple]="true">
                            <soldr-filter-item *ngFor="let item of filterItems" [label]="item.label" [value]="item.value">
                            </soldr-filter-item>
                        </soldr-filter>
                    </soldr-grid>
                `,
                {
                    data,
                    filterItems: data.map((item) => ({ label: item.id, value: item.id }))
                }
            );
            fixture.debugElement.query(By.css('.grid__button_show-filters')).nativeElement.click();
            fixture.detectChanges();

            expect(fixture.debugElement.queryAll(By.css('soldr-filter')).length).toBe(1);
            expect(fixture.debugElement.query(By.css('.grid__button_reset-filters'))).toBeDefined();
        });

        it('can reset filter values on click "Reset Filtration" button', () => {
            const fixture = MockRender<GridComponent>(
                `
                    <soldr-grid [data]="data" (resetFiltration)="onResetFiltration()">
                        <soldr-column field="id" [headerName]="'ID'"></soldr-column>
                        <soldr-column field="value" [headerName]="'Value'"></soldr-column>
                        <soldr-filter
                            [field]="'id'"
                            [title]="'Filter By Id'"
                            [multiple]="true">
                            <soldr-filter-item *ngFor="let item of filterItems" [label]="item.label" [value]="item.value">
                            </soldr-filter-item>
                        </soldr-filter>
                    </soldr-grid>
                `,
                {
                    data,
                    filterItems: data.map((item) => ({ label: item.id, value: item.id }))
                }
            );
            const component = fixture.componentInstance;
            ngMocks.stubMember(component, 'onResetFiltration', jest.fn());
            fixture.debugElement.query(By.css('.grid__button_show-filters')).nativeElement.click();
            fixture.detectChanges();
            fixture.debugElement.query(By.css('.grid__button_reset-filters')).nativeElement.click();

            expect(component.onResetFiltration).toBeCalled();
        });

        it('should display count filters if they are set', () => {
            const fixture = MockRender<GridComponent>(
                `
                    <soldr-grid [data]="data" [filtration]="filtration">
                        <soldr-column field="id" [headerName]="'ID'"></soldr-column>
                        <soldr-column field="value" [headerName]="'Value'"></soldr-column>
                        <soldr-filter
                            [field]="'id'"
                            [title]="'Filter By Id'"
                            [multiple]="true">
                            <soldr-filter-item *ngFor="let item of filterItems" [label]="item.label" [value]="item.value">
                            </soldr-filter-item>
                        </soldr-filter>
                    </soldr-grid>
                `,
                {
                    data,
                    filtration: [{ field: 'id', value: 1 }],
                    filterItems: data.map((item) => ({ label: item.id, value: item.id }))
                }
            );

            expect(
                fixture.debugElement
                    .query(By.css('.grid__button_show-filters .mc-badge'))
                    .nativeElement.textContent.trim()
            ).toBe('1');
        });
    });

    describe('Search', () => {
        const searchValue = 'searchValue';
        it('should be set if pass via property', async () => {
            const fixture = MockRender<GridComponent>(
                `
                    <soldr-grid [data]="data" [searchString]="searchValue"></soldr-grid>
                `,
                { data, searchValue }
            );
            await fixture.whenStable();

            expect(fixture.debugElement.query(By.css('.grid__search-input input')).nativeElement.value).toBe(
                searchValue
            );
        });
        it('should call filtration if value are inputted in text field', async () => {
            const searchValue = 'searchValue';
            const fixture = MockRender<GridComponent>(
                `
                    <soldr-grid [data]="data" (search)="onSearch($event)"></soldr-grid>
                `,
                { data, searchValue }
            );
            await fixture.whenStable();
            const component = fixture.componentInstance;
            ngMocks.stubMember(component, 'onSearch', jest.fn());

            const input = fixture.debugElement.query(By.css('.grid__search-input .mc-input')).nativeElement;
            fixture.point.componentInstance.searchValue = searchValue;
            input.value = searchValue;
            input.dispatchEvent(new KeyboardEvent('input'));
            fixture.detectChanges();
            input.dispatchEvent(new KeyboardEvent('keyup', { key: 'Enter' }));
            fixture.detectChanges();

            expect(fixture.point.componentInstance.searchValue).toBe(searchValue);
            expect(component.onSearch).toHaveBeenCalledWith(searchValue);
        });
        it('should reset filtration by search value on click "Reset" button', async () => {
            const searchValue = 'searchValue';
            const fixture = MockRender<GridComponent>(
                `
                    <soldr-grid [data]="data" [searchString]="searchValue" (search)="onSearch($event)"></soldr-grid>
                `,
                { data, searchValue }
            );
            const component = fixture.componentInstance;
            ngMocks.stubMember(component, 'onSearch', jest.fn());
            await fixture.whenStable();
            fixture.detectChanges();

            const cleaner = fixture.debugElement.query(By.css('.mc-cleaner')).nativeElement;
            cleaner.click();
            fixture.detectChanges();

            expect(fixture.point.componentInstance.searchValue).toBe(null);
            expect(component.onSearch).toHaveBeenCalledWith(undefined);
        });
    });

    describe('Sorting', () => {
        it('should sort by id | asc', async () => {
            const fixture = MockRender<GridComponent>(
                `
                    <soldr-grid [data]="data" (sortChanged)="onSort($event)">
                        <soldr-column field="id" [headerName]="'ID'" [sortable]="true"></soldr-column>
                        <soldr-column field="value" [headerName]="'Value'"></soldr-column>
                    </soldr-grid>
                `,
                {
                    data
                }
            );
            const component = fixture.componentInstance;
            ngMocks.stubMember(component, 'onSort', jest.fn());
            await fixture.whenStable();
            const headers = fixture.debugElement.queryAll(By.css('.ag-header-cell-label'));

            headers[0].nativeElement.click();
            await fixture.whenStable();

            expect(component.onSort).toHaveBeenCalledWith({ prop: 'id', order: SortingDirection.ASC });
        });

        it('should sort by id | desc', async () => {
            const fixture = MockRender<GridComponent>(
                `
                    <soldr-grid [data]="data" (sortChanged)="onSort($event)">
                        <soldr-column field="id" [headerName]="'ID'" [sortable]="true"></soldr-column>
                        <soldr-column field="value" [headerName]="'Value'"></soldr-column>
                    </soldr-grid>
                `,
                {
                    data
                }
            );
            const component = fixture.componentInstance;
            ngMocks.stubMember(component, 'onSort', jest.fn());
            await fixture.whenStable();
            const headers = fixture.debugElement.queryAll(By.css('.ag-header-cell-label'));

            headers[0].nativeElement.click();
            headers[0].nativeElement.click();
            await fixture.whenStable();

            expect(component.onSort).toHaveBeenCalledWith({ prop: 'id', order: SortingDirection.DESC });
        });

        it('should reset sorting for third time', async () => {
            const fixture = MockRender<GridComponent>(
                `
                    <soldr-grid [data]="data" (sortChanged)="onSort($event)">
                        <soldr-column field="id" [headerName]="'ID'" [sortable]="true"></soldr-column>
                        <soldr-column field="value" [headerName]="'Value'"></soldr-column>
                    </soldr-grid>
                `,
                {
                    data
                }
            );
            const component = fixture.componentInstance;
            ngMocks.stubMember(component, 'onSort', jest.fn());
            await fixture.whenStable();
            const headers = fixture.debugElement.queryAll(By.css('.ag-header-cell-label'));

            headers[0].nativeElement.click();
            headers[0].nativeElement.click();
            headers[0].nativeElement.click();
            await fixture.whenStable();

            expect(component.onSort).toHaveBeenCalledWith({});
        });

        it('should be set if pass via property', async () => {
            const fixture = MockRender<GridComponent>(
                `
                    <soldr-grid [data]="data" [sorting]="sorting" (sortChanged)="onSort($event)">
                        <soldr-column field="id" [headerName]="'ID'" [sortable]="true"></soldr-column>
                        <soldr-column field="value" [headerName]="'Value'"></soldr-column>
                    </soldr-grid>
                `,
                { data, sorting: { prop: 'id', order: SortingDirection.DESC } }
            );
            const component = fixture.componentInstance;
            ngMocks.stubMember(component, 'onSort', jest.fn());
            await fixture.whenStable();

            expect(component.onSort).toHaveBeenCalledWith({ prop: 'id', order: SortingDirection.DESC });
        });
    });

    describe('Scrolling', () => {
        it('should request next page if scroll to end', async () => {
            const fixture = MockRender<GridComponent>(
                `
                    <soldr-grid [data]="data" (nextPage)="onLoadNextPage()">
                        <soldr-column field="id" [headerName]="'ID'"></soldr-column>
                        <soldr-column field="value" [headerName]="'Value'"></soldr-column>
                    </soldr-grid>
                `,
                { data }
            );
            const component = fixture.componentInstance;
            ngMocks.stubMember(component, 'onLoadNextPage', jest.fn());
            await fixture.whenStable();
            const viewport = fixture.debugElement.query(By.css('.ag-body-viewport'));

            viewport.nativeElement.dispatchEvent(new Event('scroll'));
            await fixture.whenStable();

            expect(component.onLoadNextPage).toBeCalled();
        });
    });
});
