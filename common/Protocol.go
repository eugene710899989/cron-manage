package common

import (
	"context"
	"encoding/json"
	"github.com/gorhill/cronexpr"
	"strings"
	"time"
)

type Job struct {
	Name     string `json:"name"`
	Command  string `json:"command"`
	CronExpr string `json:"cron_expr"`
}

type Event struct {
	EventType int // SAVE ,DELETE
	Job       *Job
}

//任务调度计划
type JobSchedulePlan struct {
	*Job
	Expr     *cronexpr.Expression
	NextTime time.Time
}

//任务执行状态
type JobExecuteInfo struct {
	Job           *Job
	PlanTime      time.Time
	ActualTIme    time.Time
	CancelContext context.Context
	CancelFunc    context.CancelFunc
}

type JobExecuteResult struct {
	ExecuteInfo *JobExecuteInfo //执行状态
	Output      []byte          //脚本输出
	Err         error           //脚本错误
	StartTime   time.Time
	EndTime     time.Time
}

//任务执行日志
type JobLog struct {
	JobName      string `bson:"jobName"`
	Command      string `bson:"command"`
	Err          string `bson:"err"`
	Output       string `bson:"output"`
	PlanTime     int64  `bson:"planTime"`     //计划开始时间
	ScheduleTime int64  `bson:"scheduleTime"` //实际调度时间
	StartTime    int64  `bson:"startTimes"`   //任务开始时间
	EndTime      int64  `bson:"entTime"`      //任务执行时间
}

func UnPackJob(value []byte, job *Job) (err error) {
	if err = json.Unmarshal(value, job); err != nil {
		return
	}
	return
}

// 从etcd key提取任务名称
func ExtractJobName(jobKey string) string {
	return strings.TrimPrefix(jobKey, JOB_PREFIX)
}

// 从etcd key提取worker ip
func ExtractWorkIp(workerKey string) string {
	return strings.TrimPrefix(workerKey, WORKER_PREFIX)
}

// 从killer key提取任务名称
func ExtractKillerName(jobKey string) string {
	return strings.TrimPrefix(jobKey, KILLER_PREFIX)
}

func BuildEvent(eventType int, job *Job) *Event {
	return &Event{EventType: eventType, Job: job}
}

//根据时间获取任务下次执行时间 执行计划
func (job *Job) BuildJobSchedulePlan() (plan *JobSchedulePlan, err error) {
	var (
		expr *cronexpr.Expression
	)

	if expr, err = cronexpr.Parse(job.CronExpr); err != nil {
		return plan, err
	}
	return &JobSchedulePlan{job, expr, expr.Next(time.Now())}, nil

}

func (plan *JobSchedulePlan) BuildJobExecuteInfo() *JobExecuteInfo {
	ctxWithCancel, cancelFunc := context.WithCancel(context.TODO())
	return &JobExecuteInfo{Job: plan.Job, PlanTime: plan.NextTime, ActualTIme: time.Now(), CancelFunc: cancelFunc, CancelContext: ctxWithCancel}
}
