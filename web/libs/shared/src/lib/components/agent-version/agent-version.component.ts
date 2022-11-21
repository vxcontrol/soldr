import { Component, Input, OnChanges, SimpleChanges } from '@angular/core';
import { ThemePalette } from '@ptsecurity/mosaic/core';

import { Agent, canUpgradeAgent, isUpgradeAgentInProgress } from '@soldr/models';

import { AgentVersionPipe } from '../../pipes';

@Component({
    selector: 'soldr-agent-version',
    templateUrl: './agent-version.component.html',
    styleUrls: ['./agent-version.component.scss']
})
export class AgentVersionComponent implements OnChanges {
    @Input() agent: Agent;
    @Input() latestVersion: string;

    hasNewVersion: boolean;
    isRunning: boolean;
    themePalette = ThemePalette;
    agentVersionText: string;

    constructor(private versionPipe: AgentVersionPipe) {}

    ngOnChanges({ agent }: SimpleChanges): void {
        if (agent?.currentValue) {
            this.isRunning = isUpgradeAgentInProgress(this.agent);
            this.agentVersionText = this.isRunning
                ? `${this.versionPipe.transform(this.agent.version)} → ${this.versionPipe.transform(
                      this.agent.details.upgrade_task.version
                  )}`
                : `${this.versionPipe.transform(this.agent.version)}`;
        }

        this.hasNewVersion = canUpgradeAgent(this.agent, this.latestVersion);
    }
}
