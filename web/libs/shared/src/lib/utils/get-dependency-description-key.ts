import { DependencyType } from '@soldr/api';
import { EntityDependency } from '@soldr/shared';

export function getDependencyDescriptionKey(dependency: EntityDependency): string {
    switch (dependency.type) {
        case DependencyType.ToMakeAction:
            return 'shared.Shared.DependenciesView.Text.DescriptionToMakeAction';
        case DependencyType.ToReceiveData:
            return 'shared.Shared.DependenciesView.Text.DescriptionToReceiveData';
        case DependencyType.ToSendData:
            return 'shared.Shared.DependenciesView.Text.DescriptionToSendData';
        case DependencyType.AgentVersion:
            return 'shared.Shared.DependenciesView.Text.DescriptionAgentVersion';
        default:
            return '';
    }
}
