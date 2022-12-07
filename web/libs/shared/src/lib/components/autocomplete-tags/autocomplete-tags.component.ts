import {
    AfterViewInit,
    ChangeDetectorRef,
    Component,
    ElementRef,
    forwardRef,
    Input,
    TemplateRef,
    ViewChild
} from '@angular/core';
import { ControlValueAccessor, NG_VALUE_ACCESSOR, ValidationErrors, ValidatorFn, FormControl } from '@angular/forms';
import { McAutocomplete, McAutocompleteSelectedEvent } from '@ptsecurity/mosaic/autocomplete';
import { McTagInputEvent, McTagList } from '@ptsecurity/mosaic/tags';
import { concat, map, mapTo, merge, Observable, of, timer } from 'rxjs';

import { DELAY_HIDE_TOOLTIP } from '../../utils';

const TAG_MAX_LENGTH = 30;
const MAX_LIMIT_TAGS = 20;
const REGEX_TAG_NAME = /^[а-яА-ЯёЁa-zA-Z\d_\-]+$/g;
const REGEX_REPLACE_TAG_NAME = /[^а-яА-ЯёЁa-zA-Z\d_\-]+/g;

@Component({
    selector: 'soldr-autocomplete-tags',
    templateUrl: './autocomplete-tags.component.html',
    styleUrls: ['./autocomplete-tags.component.scss'],
    providers: [
        {
            provide: NG_VALUE_ACCESSOR,
            useExisting: forwardRef(() => AutocompleteTagsComponent),
            multi: true
        }
    ]
})
export class AutocompleteTagsComponent implements AfterViewInit, ControlValueAccessor {
    @Input() allTags: string[] = [];
    @Input() tagMask?: RegExp;
    @Input() tagMaskForReplace?: RegExp;
    @Input() tagMaskTooltipText?: string;

    @ViewChild('tagList', { static: false }) tagList: McTagList;
    @ViewChild('tagInput', { static: false }) tagInput: ElementRef<HTMLInputElement>;
    @ViewChild('autocomplete', { static: false }) autocomplete: McAutocomplete;

    @ViewChild('errorsTooltipTagName', { static: false }) errorsTooltipTagName: TemplateRef<any>;
    @ViewChild('errorsTooltipMaxlength', { static: false }) errorsTooltipMaxlength: TemplateRef<any>;

    control = new FormControl<any>('', [this.maxLimitTagsValidator()]);
    filteredTags: any;
    filteredTagsByInput: string[] = [];
    selectedTags: string[] = [];
    showErrorsTooltip$: Observable<boolean>;
    errorsTooltip: TemplateRef<any>;
    disabled: boolean;
    tags: string[];

    constructor(private changeDetectorRef: ChangeDetectorRef) {}

    onChange(_: string[]) {}

    writeValue(obj: string[]): void {
        this.selectedTags = obj;
    }

    registerOnChange(fn: any): void {
        this.onChange = fn;
    }

    registerOnTouched(fn: any): void {}

    setDisabledState?(isDisabled: boolean): void {
        this.disabled = isDisabled;
    }

    ngAfterViewInit(): void {
        this.filteredTags = merge(
            this.tagList.tagChanges.asObservable().pipe(
                map((selectedTags) => {
                    const values = selectedTags.map((tag: any) => tag.value);

                    return this.allTags.filter((tag) => !values.includes(tag));
                })
            ),
            this.control.valueChanges.pipe(
                map((value) => {
                    const typedText: string = value && value.new ? value.value : value;

                    this.filteredTagsByInput = typedText ? this.filter(typedText) : this.allTags.slice();

                    return this.filteredTagsByInput.filter((tag) => !this.selectedTags.includes(tag));
                })
            )
        );

        this.changeDetectorRef.detectChanges();
    }

    onInput(event: any) {
        const value = event.target.value;

        if (value && !value.match(this.tagMask || REGEX_TAG_NAME)) {
            const newValue = value.replace(this.tagMaskForReplace || REGEX_REPLACE_TAG_NAME, '') as string;
            this.processingWarnValidation(newValue, this.errorsTooltipTagName);
        }

        if (value?.length > TAG_MAX_LENGTH) {
            const newValue = value.slice(0, TAG_MAX_LENGTH) as string;
            this.processingWarnValidation(newValue, this.errorsTooltipMaxlength);
        }
    }

    addOnBlurFunc(event: FocusEvent) {
        const target: HTMLElement = event.relatedTarget as HTMLElement;

        if (!target || target.tagName !== 'MC-OPTION') {
            const mcTagEvent: McTagInputEvent = {
                input: this.tagInput.nativeElement,
                value: this.tagInput.nativeElement.value
            };

            if (this.control.valid) {
                this.onCreate(mcTagEvent);
            }
        }
    }

    onCreate(event: McTagInputEvent): void {
        const input = event.input;
        const value = event.value;
        const trimmedValue = (value || '').trim();

        if (trimmedValue) {
            const isOptionSelected = this.autocomplete.options.some((option) => option.selected);
            if (!isOptionSelected) {
                this.selectedTags = [...this.selectedTags, trimmedValue];
            }
        }

        if (input) {
            input.value = '';
        }

        this.control.setValue(null);

        if (trimmedValue) {
            this.onChange(this.selectedTags);
        }
    }

    onSelect(event: McAutocompleteSelectedEvent): void {
        event.option.deselect();

        if (event.option.value.new) {
            this.selectedTags = [...this.selectedTags, event.option.value.value];
        } else {
            this.selectedTags = [...this.selectedTags, event.option.value];
        }
        this.tagInput.nativeElement.value = '';
        this.control.setValue(null);
        this.onChange(this.selectedTags);
    }

    onRemove(removedTag: any): void {
        this.selectedTags = this.selectedTags.filter((tag) => tag !== removedTag);
        this.onChange(this.selectedTags);
        this.control.updateValueAndValidity();
    }

    clearAll() {
        this.control.setValue(null);
        this.selectedTags = [];
    }

    private processingWarnValidation(newInputValue: string, tooltipTemplate: TemplateRef<any>) {
        this.control.setValue(newInputValue);
        this.tagInput.nativeElement.value = newInputValue;

        this.errorsTooltip = tooltipTemplate;
        // eslint-disable-next-line @typescript-eslint/no-unsafe-argument
        this.showErrorsTooltip$ = concat(of(true), timer(DELAY_HIDE_TOOLTIP).pipe(mapTo(false)));
    }

    private filter(value: string): string[] {
        const filterValue = value.toLowerCase();

        return this.allTags.filter((tag) => tag.toLowerCase().indexOf(filterValue) === 0);
    }

    private maxLimitTagsValidator(): ValidatorFn {
        return (): ValidationErrors | null =>
            this.selectedTags?.length > MAX_LIMIT_TAGS ? { maxLimitTags: true } : null;
    }
}
