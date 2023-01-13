package store

import "fmt"

type TaskStatus string

const (
	TaskStatusNew     TaskStatus = "new"
	TaskStatusRunning TaskStatus = "running"
	TaskStatusReady   TaskStatus = "ready"
	TaskStatusFailed  TaskStatus = "failed"

	DBUpgradeTasksTableName      = "upgrade_tasks" // nolint: unused
	DBUpgradeTasksIDColName      = "id"
	DBUpgradeTasksAgentIDColName = "agent_id"
	DBUpgradeTasksVersionColName = "version"

	DBAgentsTableName   = "agents" // nolint: unused
	DBAgentsIDColName   = "id"     // nolint: unused
	DBAgentsHashColName = "hash"
)

type Task struct {
	ID        string
	AgentID   string
	AgentHash string
	Version   string
}

func (t *Task) String() string {
	return fmt.Sprintf("Task ID: %s, Agent ID: %s, Agent hash: %s, Version: %s", t.ID, t.AgentID, t.AgentHash, t.Version)
}

func ParseRowIntoTask(r map[string]string) (*Task, error) { // nolint: unused
	var err error
	t := &Task{}
	t.ID, err = getUpgradeTaskColValue(r, DBUpgradeTasksIDColName)
	if err != nil {
		return nil, err
	}
	t.AgentID, err = getUpgradeTaskColValue(r, DBUpgradeTasksAgentIDColName)
	if err != nil {
		return nil, err
	}
	t.AgentHash, err = getUpgradeTaskColValue(r, DBAgentsHashColName)
	if err != nil {
		return nil, err
	}
	t.Version, err = getUpgradeTaskColValue(r, DBUpgradeTasksVersionColName)
	if err != nil {
		return nil, err
	}
	return t, nil
}

func getUpgradeTaskColValue(r map[string]string, colName string) (string, error) {
	f, ok := r[colName]
	switch {
	case !ok:
		return "", fmt.Errorf("failed to get the upgrade task's %s: the column not found in the passed row", colName)
	case len(f) == 0:
		return "", fmt.Errorf("failed to get the upgrade task's %s: the fetched value is empty", colName)
	default:
		return f, nil
	}
}
