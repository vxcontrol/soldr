import { Component, EventEmitter, Inject, Input, OnChanges, Output, SimpleChanges } from '@angular/core';
import { ThemePalette } from '@ptsecurity/mosaic/core';
import { BehaviorSubject, pairwise, Subscription, take } from 'rxjs';

import { PERMISSIONS_TOKEN } from '@soldr/core';
import { Agent, AgentUpgradeTask, canUpgradeAgent, isUpgradeAgentInProgress } from '@soldr/models';

import { ProxyPermission } from '../../types';

const LATEST_VERSION = 'latest';

@Component({
    selector: 'soldr-upgrade-status-message',
    templateUrl: './upgrade-status-message.component.html',
    styleUrls: ['./upgrade-status-message.component.scss']
})
export class UpgradeStatusMessageComponent implements OnChanges {
    @Input() agent: Agent;
    @Input() latestBinaryVersion: string;
    @Input() isUpgradingAgents: boolean;
    @Input() isCancelUpgradingAgent: boolean;

    @Output() upgradeAgents = new EventEmitter<{ agents: Agent[]; version: string }>();
    @Output() cancelUpgradeAgent = new EventEmitter<{ hash: string; task: AgentUpgradeTask }>();
    @Output() refresh = new EventEmitter();

    canShowFailure = false;
    canShowInProgress = false;
    canShowNewVersion = false;
    canShowSuccess = false;
    isHidden = false;
    subscription = new Subscription();
    task: AgentUpgradeTask;
    themePalette = ThemePalette;

    private inputIsUpgradingAgents$ = new BehaviorSubject<boolean>(false);
    private inputIsCancelUpgradingAgent$ = new BehaviorSubject<boolean>(false);
    readonly latestVersion = LATEST_VERSION;

    constructor(@Inject(PERMISSIONS_TOKEN) public permitted: ProxyPermission) {}

    ngOnChanges({ agent, isUpgradingAgents, latestBinaryVersion, isCancelUpgradingAgent }: SimpleChanges): void {
        if (agent?.currentValue || latestBinaryVersion?.currentValue) {
            this.isHidden = false;
            this.canShowInProgress = isUpgradeAgentInProgress(this.agent);
            this.canShowNewVersion = canUpgradeAgent(this.agent, this.latestBinaryVersion);
            this.task = this.agent?.details?.upgrade_task;
            this.canShowFailure =
                !this.canShowNewVersion && this.task?.status === 'failed' && this.task?.reason !== 'Canceled.By.User';
            this.canShowSuccess = !this.canShowFailure && this.task?.status === 'ready';
        }

        if (isUpgradingAgents) {
            this.inputIsUpgradingAgents$.next(isUpgradingAgents.currentValue as boolean);
        }

        if (isCancelUpgradingAgent) {
            this.inputIsCancelUpgradingAgent$.next(isCancelUpgradingAgent.currentValue as boolean);
        }
    }

    hide() {
        this.isHidden = true;
    }

    upgrade() {
        this.inputIsUpgradingAgents$.pipe(pairwise(), take(2)).subscribe(([oldValue, newValue]) => {
            if (oldValue && !newValue) {
                this.refresh.emit();
            }
        });
        this.upgradeAgents.emit({ agents: [this.agent], version: this.latestBinaryVersion });
    }

    cancel() {
        this.inputIsCancelUpgradingAgent$.pipe(pairwise(), take(2)).subscribe(([oldValue, newValue]) => {
            if (oldValue && !newValue) {
                this.refresh.emit();
            }
        });
        this.cancelUpgradeAgent.emit({ hash: this.agent.hash, task: this.task });
    }

    repeat() {
        this.upgrade();
    }
}
