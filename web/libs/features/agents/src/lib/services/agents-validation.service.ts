import { Injectable } from '@angular/core';
import { catchError, filter, from, map, Observable, of, switchMap, toArray } from 'rxjs';

import { AgentsService, allListQuery, ModelsAgent, PrivateAgents, SuccessResponse } from '@soldr/api';

@Injectable({
    providedIn: 'root'
})
export class AgentsValidationService {
    constructor(private agentsService: AgentsService) {}

    getIsExistedAgentsByDescription(description: string, exclude: string[]): Observable<boolean> {
        const query = allListQuery({ filters: [{ field: 'description', value: [description] }] });

        return this.agentsService.fetchList(query).pipe(
            switchMap((response: SuccessResponse<PrivateAgents>) => from(response.data?.agents)),
            filter((agent: ModelsAgent) => !exclude.includes(agent.description)),
            toArray(),
            map((agents) => agents.length > 0),
            catchError(() => of(false))
        );
    }
}
