import { Pipe, PipeTransform } from '@angular/core';

import { ModelsSemVersion } from '@soldr/api';

@Pipe({
    name: 'moduleVersion'
})
export class ModuleVersionPipe implements PipeTransform {
    transform(version: ModelsSemVersion): string {
        return version ? `${version.major}.${version.minor}.${version.patch}` : '';
    }
}
