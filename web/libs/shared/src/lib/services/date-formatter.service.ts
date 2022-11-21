import { Injectable } from '@angular/core';

import {
    AbsoluteLongDateTimeWithSecondsPipe,
    LanguageService,
    RelativeLongDateTimePipe,
    ToDateTimePipe
} from '@soldr/shared';

@Injectable({
    providedIn: 'root'
})
export class DateFormatterService {
    constructor(
        private toDateTimePipe: ToDateTimePipe,
        private relativeDatePipe: RelativeLongDateTimePipe,
        private absoluteLongDateTimeWithSeconds: AbsoluteLongDateTimeWithSecondsPipe,
        private languageService: LanguageService
    ) {}

    formatToRelativeDate(data: any[], dateField: string): any[] {
        return data.map((item) => ({
            ...item,
            [dateField]: this.relativeDatePipe.transform(this.toDateTime(item[dateField] as string))
        }));
    }

    formatToAbsoluteLongWithSeconds(data: any[], dateField: string): any[] {
        return data.map((item) => ({
            ...item,
            [dateField]: this.absoluteLongDateTimeWithSeconds.transform(this.toDateTime(item[dateField] as string))
        }));
    }

    private toDateTime(date: string) {
        return this.toDateTimePipe.transform(date, this.languageService.lang);
    }
}
