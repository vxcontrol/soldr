import { AfterViewInit, Component, ElementRef, EventEmitter, Output } from '@angular/core';
import { ControlValueAccessor, NG_VALUE_ACCESSOR } from '@angular/forms';
import {
    RefresherValue,
    defaultRefresherOptions,
    defaultRefresherValue
} from '@mosaic-design/infosec-components/components/refresher';
import { OnChange, OnTouch } from '@mosaic-design/infosec-components/types';

@Component({
    selector: 'soldr-refresher',
    templateUrl: './refresher.component.html',
    styleUrls: ['./refresher.component.scss'],
    providers: [
        {
          provide: NG_VALUE_ACCESSOR,
          useExisting: RefresherComponent,
          multi: true
        }
    ]
})
export class RefresherComponent implements ControlValueAccessor, AfterViewInit {
    defaultRefresherOptions = defaultRefresherOptions;
    value = defaultRefresherValue;

    @Output() refresh = new EventEmitter();

    constructor(private element: ElementRef) {
    }

    ngAfterViewInit(): void {
      this.element.nativeElement.querySelector('[mc-icon="pt-icons-refresh_16"]').classList.replace('pt-icons', 'soldr-icons');
      this.element.nativeElement.querySelector('[mc-icon="pt-icons-refresh_16"]').classList.replace('pt-icons-refresh_16', 'soldr-icons-refresh_16');
    }

    onChange: OnChange<RefresherValue> = () => {};
    onTouch: OnTouch = () => {};

    registerOnChange(fn: OnChange<RefresherValue>): void {
        this.onChange = fn;
    }

    registerOnTouched(fn: OnTouch): void {
        this.onTouch = fn;
    }

    writeValue(value: RefresherValue): void {
        this.value = value;
    }
}
