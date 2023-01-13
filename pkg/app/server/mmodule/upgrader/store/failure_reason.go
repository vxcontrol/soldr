package store

type TaskFailureReason string

const (
	TaskFailureReasonTimeout                TaskFailureReason = "Timeout"
	TaskFailureReasonUnexpectedVersion      TaskFailureReason = "Unexpected.Version"
	TaskFailureReasonChanClosedUnexpectedly TaskFailureReason = "Watcher.Failed"
	TaskFailureReasonUpgraderFailed         TaskFailureReason = "Upgrader.Failed"
)
