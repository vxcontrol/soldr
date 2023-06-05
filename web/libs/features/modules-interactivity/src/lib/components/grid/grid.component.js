import Vue from 'vue';

import { DraggableSelect } from '../draggable-select/draggable-select.components';

// noinspection HtmlUnknownAttribute
const template = `
<!--suppress AngularInvalidAnimationTriggerAssignment -->
</template>
    <div class="vue-grid__wrapper layout-column" ref="wrapper">
        <div class="flex-none layout-margin-bottom-xl" ref="filter">
            <div class="layout-row">
                <div v-if="hasSearch" class="flex-auto layout-row layout-align-start-center">

                    <div class="vue-grid__rows-selector" v-if="canSelect && multiple && showSelectionMenu">
                        <el-checkbox
                            v-model="isSelectedAll"
                            :indeterminate="selected.length > 0 && !isSelectedAll"
                            @change="toggleAllSelection()">
                        </el-checkbox>

                        <el-dropdown trigger="click" placement="bottom-start" @command="onSelectionCommand">
                            <span class="el-dropdown-link">
                                <i class="el-icon-arrow-down el-icon--right"></i>
                            </span>
                            <el-dropdown-menu slot="dropdown">
                                <el-dropdown-item command="selectAll">
                                    {{ $t('ModulesInteractivity.Grid.DropdownItemText.SelectAll') }}
                                </el-dropdown-item>
                                <el-dropdown-item command="selectAllOnPage">
                                    {{ $t('ModulesInteractivity.Grid.DropdownItemText.SelectPage') }}
                                </el-dropdown-item>
                                <el-dropdown-item command="clearSelection">
                                    {{ $t('ModulesInteractivity.Grid.DropdownItemText.ClearSelection') }}
                                </el-dropdown-item>
                            </el-dropdown-menu>
                        </el-dropdown>
                    </div>

                    <div class="flex-auto layout-row layout-align-space-between"
                         :class="{ 'vue-grid__search-container_has-searchable-fields': hasSelectSearchableFields }">
                        <el-select
                            v-if="hasSelectSearchableFields"
                            class="vue-grid__search-fields flex-none"
                            v-model="selectedSearchField"
                            value-key="prop"
                            slot="prepend"
                            @change="onSelectColumn()">
                            <el-option
                                v-for="column in searchableColumns"
                                :key="column.value"
                                :label="column.label"
                                :value="column">
                            </el-option>
                        </el-select>

                        <el-input
                            v-if="isSearchByText"
                            class="vue-grid__search-input flex-auto"
                            v-model="searchValue"
                            :placeholder="currentSearchPlaceholder"
                            @change="search()">
                        </el-input>

                        <el-select
                            ref="searchValuesSelect"
                            v-if="isSearchBySelect"
                            v-model="searchValue"
                            class="vue-grid__search-select flex-auto"
                            :multiple="selectedSearchField.search.multiple"
                            :placeholder="currentSearchPlaceholder"
                            @clear="search()">
                            <el-option
                                v-for="item in selectedSearchField.search.items"
                                :key="item.value"
                                :label="item.label"
                                :value="item.value"
                                @change="search()">
                            </el-option>
                        </el-select>

                        <el-button
                            class="vue-grid__search-button flex-none"
                            size="medium"
                            icon="el-icon-search"
                            @click="search()">
                        </el-button>
                    </div>
                </div>

                <div v-bind:class="{'layout-margin-left-xl': $slots.toolbar, 'flex-auto': !hasSearch, 'flex-none': hasSearch }">
                    <slot name="toolbar" v-bind:selected="selected" v-bind:isSelectedAll="isSelectedAll"></slot>
                </div>

                <div v-if="hasSettings" class="flex-none layout-margin-left-xl">
                    <el-tooltip :content="$t('ModulesInteractivity.Grid.TooltipText.ColumnsSettings')">
                        <el-button icon="el-icon-setting" v-popover:columnsSelectorPopover></el-button>
                    </el-tooltip>
                </div>
            </div>
        </div>

        <div
            class="vue-grid__container flex-auto"
            :class="{ 'vue-grid_no-selectable': noSelectionText }">

            <data-tables-server
                ref="table"
                :loading="isLoading"
                :data="data"
                :page-size="parseInt(query.pageSize || 50)"
                :current-page="parseInt(query.page || 1)"
                :table-props="tableProps"
                :row-class-name="calcTableRowClassName"
                :pagination-props="{
                    layout: 'slot, prev, pager, next, sizes',
                    pageSizes: [10, 30, 50],
                    total: this.total
                }"
                @query-change="loadData"
                @row-click="onRowClick"
                @selection-change="onSelectionChange"
                @header-dragend="saveSettings">

                <template v-slot:empty>
                    <template v-if="$slots['custom-no-rows']">
                        <slot name="custom-no-rows"></slot>
                    </template>
                    <template v-else>
                        {{ searchValue ? $t('Common.Pseudo.Text.NotFound') : emptyText || $t('Common.Pseudo.Text.NoData') }}
                    </template>
                </template>

                <template v-if="renderComponent">
                    <slot v-for="column in visibleColumns" :name="'column-' + column.prop"></slot>
                </template>

                <template v-slot:pagination v-if="canSelect">
                    <span class="layout-row">
                        <div class="el-pagination__selected">
                            {{
                                multipleSelection ? $t('ModulesInteractivity.Grid.Text.RowsSummarySelected', {
                                    selected: isSelectedAll ? 'all' : selected.length,
                                    total: total
                                }) : $t('ModulesInteractivity.Grid.Text.RowsSummary', {
                                    total: total
                                })
                            }}
                        </div>
                    </span>
                </template>
            </data-tables-server>

            <el-popover
                ref="columnsSelectorPopover"
                placement="bottom-end"
                width="640"
                trigger="click"
                :visible-arrow="false">
                <div class="layout-padding-l">
                    <draggable-select
                        class="layout-fill_horizontal"
                        v-model="visibleColumns"
                        multiple
                        value-key="prop"
                        :placeholder="$t('ModulesInteractivity.Grid.SelectPlaceholder.SelectColumns')"
                        @change="saveSettings">
                        <el-option
                            v-for="item in availableColumns"
                            :key="item.prop"
                            :label="item.label"
                            :value="item">
                        </el-option>
                    </draggable-select>
                </div>
            </el-popover>
        </div>

        <div class="flex-none" ref="footerText">
            {{ footerText }}
        </div>
    </div>
</template>
`;

export const GridComponent = Vue.extend({
    name: 'grid',

    components: { DraggableSelect },

    props: {
        storageKey: { type: String },
        ignoreEvents: { type: Boolean },
        canSelect: { type: Boolean },
        multiple: { type: Boolean },
        multipleSelection: { type: Boolean },
        showSelectionMenu: { type: Boolean, default: true },
        query: { type: Object },
        data: { type: Array },
        total: { type: Number },
        isLoading: { type: Boolean },
        searchPlaceholder: { type: String },
        searchFilter: { type: Object },
        columnsConfig: { type: Object, default: () => ({}) },
        emptyText: { type: String },
        footerText: { type: String },
        fullTextSearch: { type: Boolean, default: true },
        noSelectionText: { type: Boolean, default: true },
        defaultSort: { type: Object },
        hasSettings: { type: Boolean, default: true },
        resizeColumns: { type: Boolean, default: false },
        hasSearch: { type: Boolean, default: true }
    },

    data() {
        return {
            componentStorageKey: 'grid',
            selectedSearchField: undefined,
            wrapperResizeObserver: undefined,
            gridColumnsResizeObserver: undefined,
            isSelectedAll: false,
            selected: [],
            searchValue: '',
            tableHeight: 0,
            pagerHeight: 32,
            resizeGridCallback: this.resizeGrid.bind(this),
            visibleColumns: [],
            renderComponent: false,
            canSaveSettings: false
        };
    },

    mounted() {
        window.addEventListener('resize', this.resizeGridCallback);

        if (this.$refs.wrapper) {
            this.wrapperResizeObserver = new ResizeObserver(this.resizeGrid);
            this.wrapperResizeObserver.observe(this.$refs.wrapper);
        }

        if (this.$refs.table) {
            this.gridColumnsResizeObserver = new ResizeObserver(this.saveSettings);
            this.gridColumnsResizeObserver.observe(this.$refs.table.$el);
        }

        if (this.searchFilter) {
            this.selectedSearchField = this.searchFilter
                ? this.searchableColumns.find((item) => item.prop === this.searchFilter.field)
                : this.searchableColumns[0];

            if (this.isSearchByText || (this.isSearchBySelect && this.selectedSearchField.search.multiple)) {
                this.searchValue = this.searchFilter.value;
            } else if (!this.selectedSearchField.search.multiple) {
                this.searchValue = this.searchFilter.value[0];
            }
        } else {
            this.selectedSearchField = this.searchableColumns[0];
        }

        this.loadSettings();
    },

    destroyed() {
        window.removeEventListener('resize', this.resizeGridCallback);

        if (this.$refs.wrapper) {
            this.wrapperResizeObserver.unobserve(this.$refs.wrapper);
        }
    },

    computed: {
        isSearchByText() {
            return this.selectedSearchField && this.selectedSearchField.search === true;
        },
        isSearchBySelect() {
            return (
                this.selectedSearchField && this.selectedSearchField.search && !!this.selectedSearchField.search.items
            );
        },
        hasSelectSearchableFields() {
            return this.searchableColumns.length > 1;
        },
        tableProps() {
            return {
                rowClassName: this.calcTableRowClassName,
                height: this.tableHeight,
                defaultSort: this.defaultSort,
                border: this.resizeColumns
            };
        },
        searchableColumns() {
            const columns = this.availableColumns.filter((column) => !!column.search);

            return [
                ...(this.fullTextSearch
                    ? [
                          {
                              label: this.$t('Common.Pseudo.Text.All'),
                              prop: 'data',
                              search: true
                          }
                      ]
                    : []),
                ...columns
            ];
        },
        currentSearchPlaceholder() {
            return !this.selectedSearchField || this.selectedSearchField.prop === 'data'
                ? this.searchPlaceholder
                : this.selectedSearchField.label;
        },
        availableColumns() {
            return Object.keys(this.$slots)
                .filter((key) => key.startsWith('column'))
                .filter((key) => !!this.columnsConfig[this.$slots[key][0].componentOptions.propsData.prop])
                .sort((a, b) => {
                    const propA = this.$slots[a][0].componentOptions.propsData.prop;
                    const propB = this.$slots[b][0].componentOptions.propsData.prop;

                    return this.columnsConfig[propA] && this.columnsConfig[propB]
                        ? this.columnsConfig[propA].index - this.columnsConfig[propB].index
                        : 0;
                })
                .map((key) => {
                    const slot = this.$slots[key];
                    const prop = slot[0].componentOptions.propsData.prop;
                    const config = this.columnsConfig[prop];

                    return {
                        prop,
                        label: slot[0].componentOptions.propsData.label,
                        search: (config && config.search) || false,
                        default: config ? config.default === true : false
                    };
                });
        }
    },

    methods: {
        // hack for force update after columns order
        forceRerender() {
            this.renderComponent = false;

            this.$nextTick(() => {
                this.renderComponent = true;
            });
        },
        resizeGrid() {
            const filterMargin = this.$refs.filter
                ? parseInt(getComputedStyle(this.$refs.filter).getPropertyValue('margin-bottom'), 10)
                : 0;
            this.tableHeight =
                !this.$refs.wrapper || !this.$refs.filter
                    ? 0
                    : Math.max(
                          0,

                          this.$refs.wrapper.clientHeight -
                              this.$refs.filter.clientHeight -
                              this.pagerHeight -
                              filterMargin -
                              this.$refs.footerText.clientHeight
                      );

            if (this.$refs.table) {
                this.$refs.table.$refs.elTable.doLayout();
            }
        },
        loadData(query) {
            if (this.ignoreEvents) {
                return;
            }

            const processedQuery = JSON.parse(JSON.stringify(query || this.$refs.table.queryInfo));

            if (!processedQuery.type) {
                processedQuery.type = 'init';
            }
            if (processedQuery.page === null) {
                processedQuery.page = 1;
            }
            if (processedQuery.pageSize === null) {
                processedQuery.pageSize = 10;
            }

            processedQuery.lang = this.$i18n.locale;
            this.$emit('query-change', processedQuery);
        },
        calcTableRowClassName({ row }) {
            return this.selected.find((item) => item.id === row.id) ? 'el-table__row_selected' : undefined;
        },
        search() {
            const value =
                this.isSearchBySelect && !this.selectedSearchField.search.multiple
                    ? this.searchValue !== ''
                        ? [this.searchValue]
                        : []
                    : this.searchValue;

            const searchProp =
                this.selectedSearchField.search && this.selectedSearchField.search.prop
                    ? this.selectedSearchField.search.prop
                    : this.selectedSearchField.prop;
            const emittedValue =
                value === '' || (Array.isArray(value) && value.length === 0)
                    ? { field: 'data', value: '' }
                    : {
                          field: searchProp,
                          value
                      };

            if (!this.visibleColumns.find((column) => column.prop === this.selectedSearchField.prop)) {
                const searchableColumn = this.availableColumns.find(
                    (column) => column.prop === this.selectedSearchField.prop
                );

                if (searchableColumn) {
                    this.visibleColumns.push(searchableColumn);
                }
            }

            if (this.isSearchByText && emittedValue) {
                emittedValue.value = emittedValue.value.trim();
            }

            this.$emit('search', emittedValue);
        },
        onSelectionChange(value) {
            this.selected = value;
            this.$emit('selection-change', this.isSelectedAll ? 'all' : value);
        },
        onRowClick(row, column, event) {
            if (!this.canSelect) {
                return false;
            }

            const elTable = this.$refs.table.$refs.elTable;

            if (event.ctrlKey && this.multiple) {
                elTable.toggleRowSelection(row);
            } else if (event.shiftKey && this.multiple) {
                const startIndex = elTable.data.indexOf(this.selected[this.selected.length - 1]);
                const endIndex = elTable.data.indexOf(row);
                const selection = elTable.data.slice(
                    Math.min(startIndex, endIndex),
                    Math.max(startIndex, endIndex) + 1
                );
                for (const selectedRow of selection) {
                    elTable.toggleRowSelection(selectedRow, true);
                }
            } else {
                elTable.clearSelection();
                elTable.toggleRowSelection(row, true);
            }

            this.isSelectedAll = this.multiple && this.selected.length === this.total;
        },
        onSelectionCommand(command) {
            if (!this.$refs.table) {
                return;
            }

            const elTable = this.$refs.table.$refs.elTable;
            const rowsOnPage = elTable.data;
            switch (command) {
                case 'selectAll':
                    this.isSelectedAll = true;
                    for (const row of rowsOnPage) {
                        elTable.toggleRowSelection(row, true);
                    }
                    break;
                case 'selectAllOnPage':
                    this.isSelectedAll = false;
                    for (const row of rowsOnPage) {
                        elTable.toggleRowSelection(row, true);
                    }
                    break;
                case 'clearSelection':
                    this.isSelectedAll = false;
                    elTable.clearSelection();
                    break;
            }
        },
        toggleAllSelection() {
            if (this.isSelectedAll) {
                this.onSelectionCommand('selectAll');
            } else {
                this.onSelectionCommand('clearSelection');
            }
        },
        onSelectColumn() {
            this.searchValue = '';
            this.$emit('search-column-change', this.selectedSearchField);
            this.search();

            // сброс выбранного значения в случае выбираемых полей при переходе между одиночным значением и множественным
            if (this.$refs.searchValuesSelect) {
                this.$refs.searchValuesSelect.selectedLabel = '';
            }
        },
        loadSettings() {
            if (!this.storageKey) {
                this.visibleColumns = this.availableColumns.filter((column) => column.default);

                return;
            }
            const storedSettings = window.localStorage.getItem(this.componentStorageKey);
            const parsedSettings = (storedSettings ? JSON.parse(storedSettings) : {})[this.storageKey] || {};

            // visibility
            this.visibleColumns = parsedSettings.visibleColumns
                ? parsedSettings.visibleColumns
                : this.availableColumns.filter((column) => column.default);

            this.$refs.table.$refs.elTable.doLayout();

            // columns sizes
            if (this.$refs.table) {
                const elTable = this.$refs.table.$children.find(
                    (item) => item.$vnode.componentOptions.tag === 'el-table'
                );
                const elTableHeader = elTable.$children.find(
                    (item) => item.$vnode.componentOptions.tag === 'table-header'
                );

                elTableHeader.$on('hook:updated', () => {
                    this.$nextTick(() => {
                        if (!parsedSettings.columnsParams) {
                            this.canSaveSettings = true;
                            return;
                        }

                        if (
                            this.visibleColumns.length > 0 &&
                            parsedSettings.columnsParams.length > 0 &&
                            elTableHeader.columns.length > 0
                        ) {
                            for (const column of elTableHeader.columns) {
                                const param = (parsedSettings.columnsParams || []).find(
                                    (item) => item.prop === column.property
                                );
                                if (param && param.width) {
                                    column.width = param.width;
                                }
                            }

                            this.$refs.table.$refs.elTable.doLayout();
                            this.canSaveSettings = true;
                            elTableHeader.$off('hook:updated');
                        }
                    });
                });
            }
        },
        saveSettings() {
            if (!this.canSaveSettings || !this.storageKey) {
                return;
            }

            let columnsParams;

            if (this.$refs.table) {
                const elTable = this.$refs.table.$children.find(
                    (item) => item.$vnode.componentOptions.tag === 'el-table'
                );
                const elTableHeader = elTable.$children.find(
                    (item) => item.$vnode.componentOptions.tag === 'table-header'
                );

                columnsParams = elTableHeader.columns.map((column) => ({
                    prop: column.property,
                    width: column.width
                }));
            }

            const componentKey = this.componentStorageKey;
            const storageKey = this.storageKey;
            const currentSettings = window.localStorage.getItem(componentKey)
                ? JSON.parse(window.localStorage.getItem(componentKey))
                : {};
            currentSettings[storageKey] = {
                visibleColumns: this.visibleColumns,
                columnsParams
            };

            window.localStorage.setItem(componentKey, JSON.stringify(currentSettings));
        },

        resetSearchFilter() {
            this.searchValue = '';
            this.selectedSearchField = this.searchableColumns[0];
        }
    },

    watch: {
        data(data) {
            const newSelection = data.filter((item) =>
                this.selected.find((selectedItem) => selectedItem.id === item.id)
            );
            setTimeout(() => {
                for (const row of newSelection) {
                    this.$refs.table.$refs.elTable.toggleRowSelection(row, true);
                }
                this.isSelectedAll =
                    (this.isSelectedAll && newSelection.length === this.$refs.table.pageSize) ||
                    (newSelection.length > 1 && newSelection.length === this.total);
                this.onSelectionChange(newSelection);
            });
        },
        visibleColumns() {
            this.forceRerender();
            this.saveSettings();
        },
        searchFilter(value, old) {
            if (!value) {
                this.resetSearchFilter();
            } else if (!old && this.searchFilter.value) {
                this.selectedSearchField = this.searchFilter
                    ? this.searchableColumns.find((item) => item.prop === this.searchFilter.field)
                    : this.searchableColumns[0];

                if (this.isSearchByText || (this.isSearchBySelect && this.selectedSearchField.search.multiple)) {
                    this.searchValue = this.searchFilter.value;
                } else if (!this.selectedSearchField.search.multiple) {
                    this.searchValue = this.searchFilter.value[0];
                }
            }
        },
        storageKey() {
            this.loadSettings();
        },
        query(value) {
            if (value && value.sort && value.sort.prop) {
                this.$refs.table.$refs.elTable.sort(value.sort.prop, value.sort.order);
            }
        }
    },

    template
});
