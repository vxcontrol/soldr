import { Component, Input, OnInit } from '@angular/core';
import { AbstractControl, FormBuilder, FormControl, FormGroup, ValidationErrors, ValidatorFn } from '@angular/forms';
import { ThemePalette } from '@ptsecurity/mosaic/core';

import { FilesTreeService } from '../../services';

@Component({
    selector: 'soldr-create-file-modal',
    templateUrl: './create-file-modal.component.html',
    styleUrls: ['./create-file-modal.component.scss']
})
export class CreateFileModalComponent implements OnInit {
    @Input() path: string;
    @Input() prefix: string;

    form: FormGroup<{ path: FormControl<string> }>;
    themePalette = ThemePalette;

    constructor(private filesTreeService: FilesTreeService, private formBuilder: FormBuilder) {}

    ngOnInit(): void {
        this.form = this.formBuilder.group({
            path: [this.path, [this.getPathClibsValidator()]]
        });
    }

    getPath(): string | undefined {
        const pathControl = this.form.get('path');
        pathControl.markAsDirty();
        pathControl.updateValueAndValidity();

        return this.form.status === 'VALID' ? pathControl.value : undefined;
    }

    private getPathClibsValidator(): ValidatorFn {
        return (control: AbstractControl): ValidationErrors | null => {
            const path = control.value as string;
            const segments = path.split('/');
            const part = segments[1];
            const pathWithoutSectionAndPart = segments.slice(2).join('/');

            return part === 'clibs' &&
                !control.pristine &&
                !this.filesTreeService.isValidClibsFilePath(pathWithoutSectionAndPart)
                ? { clibs: true }
                : null;
        };
    }
}
