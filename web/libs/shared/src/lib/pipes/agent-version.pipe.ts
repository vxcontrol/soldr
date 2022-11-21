import { Pipe, PipeTransform } from '@angular/core';

import { removeHashFromVersion } from '@soldr/models';

@Pipe({
    name: 'agentVersion'
})
export class AgentVersionPipe implements PipeTransform {
    transform(value: string): string {
        return removeHashFromVersion(value);
    }
}
