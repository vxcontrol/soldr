import { Pipe, PipeTransform } from '@angular/core';
import { TranslocoService } from '@ngneat/transloco';

@Pipe({
    name: 'lastConnected'
})
export class LastConnectedPipe implements PipeTransform {
    constructor(private transloco: TranslocoService) {}

    transform(value: string): unknown {
        return this.transloco.translate('agents.Agents.AgentsList.Text.LastConnected', {
            relativeDateTime: value.toLowerCase()
        });
    }
}
