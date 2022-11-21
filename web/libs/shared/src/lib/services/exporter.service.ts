import { Injectable } from '@angular/core';
import { DateTime } from 'luxon';

const VALUE_DELIMITER = ',';
const ROW_DELIMITER = '\r\n';

@Injectable({
    providedIn: 'root'
})
export class ExporterService {
    mapper: Record<string, (value: any) => string> = {};
    headers: Record<string, () => string> = {};

    constructor() {}

    toCsv(baseFileName: string, items: any[], columns: string[]) {
        const data = this.format(items, columns);
        const time = DateTime.now().toFormat('yyyy-MM-dd HH-mm-ssZZZ');
        const csvData = `\ufeff${data}`;
        const blob = new Blob([csvData], { type: 'data:text/csv;charset=utf-8' });
        const url = window.URL.createObjectURL(blob);
        const filename = `${baseFileName}_${time}.csv`;

        const a = document.createElement('a');

        a.setAttribute('href', url);
        a.setAttribute('download', filename);

        a.click();
    }

    private format(data: any[], columns: string[]): string {
        return [
            columns
                .map((column) => this.encodeValue(this.headers[column] ? this.headers[column]() : column))
                .join(VALUE_DELIMITER)
        ]
            .concat(
                ...data.map((item: Record<string, any>) =>
                    columns
                        .map((column) => {
                            const value = this.mapper[column] ? this.mapper[column](item) : (item[column] as string);

                            return this.encodeValue(value);
                        })
                        .join(VALUE_DELIMITER)
                )
            )
            .join(ROW_DELIMITER);
    }

    private encodeValue(value: string): string {
        if (new RegExp(`${VALUE_DELIMITER}|\r|\n|"`).test(value)) {
            return `"${value.replace(/"/g, '""')}"`;
        }

        if (!value) {
            return '""';
        }

        return value;
    }
}
