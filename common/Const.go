package common

const (
	JOB_PREFIX    = "/cron/jobs/"
	LOCKER_PREFIX = "/cron/locks/"
	KILLER_PREFIX = "/cron/killers/"
	WORKER_PREFIX = "/cron/workers/"
	EVENT_SAVE    = 1 //保存事件
	EVENT_DELETE  = 2 //删除事件
	EVENT_KILL    = 3 //强杀事件
)
