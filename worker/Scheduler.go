package worker

import (
	"cron-manage/common"
	"fmt"
	"log"
	"time"
)

//调度协程
type Scheduler struct {
	EventChan    chan *common.Event                 //任务队列
	SchedulePlan map[string]*common.JobSchedulePlan //任务调度计划表 每个任务的下次执行时间
	ExcuteInfo   map[string]*common.JobExecuteInfo  //任务调度执行表 每个任务的执行结果
	ResultChan   chan *common.JobExecuteResult      //任务调度执行结果
}

var (
	Schedule *Scheduler
)

//重新计算任务调度状态
func (schedule *Scheduler) TrySchedule() (NearestNextTime time.Duration) {
	var (
		jobPlan  *common.JobSchedulePlan
		now      time.Time
		nearTime *time.Time
	)
	now = time.Now()

	//若任务表空，返回固定配置时间间隔
	if len(schedule.SchedulePlan) == 0 {
		NearestNextTime = time.Second
		return
	}

	// 遍历所有任务，

	for _, jobPlan = range schedule.SchedulePlan {
		// 过期或到时任务立即执行
		//fmt.Println(" job ", jobPlan.Name, " time ", jobPlan.NextTime.Format("2006-01-02 15:04:05"))
		//fmt.Printf("%#v\n", jobPlan)
		if jobPlan.NextTime.Before(now) || jobPlan.NextTime.Equal(now) {
			//todo 尝试执行job plan 更新next time
			log.Println("尝试执行任务 ", jobPlan.Job.Command)
			schedule.TryStartJob(jobPlan)
			jobPlan.NextTime = jobPlan.Expr.Next(now)
		}
		if nearTime == nil || jobPlan.NextTime.Before(*nearTime) {
			//fmt.Println("near job ", jobPlan.Name, " time ", jobPlan.NextTime.Format("2006-01-02 15:04:05"))
			nearTime = &jobPlan.NextTime
		}

	}
	// 最近的要过期任务时间间隔
	NearestNextTime = nearTime.Sub(time.Now())
	log.Println("next ", NearestNextTime/time.Second, " s")
	return
}

//尝试执行任务 执行任务可能运行很久 1分钟调度60次
func (scheduler *Scheduler) TryStartJob(plan *common.JobSchedulePlan) {
	var (
		jobExecuting bool
	)
	//若任务正在执行，跳过本次调度
	if _, jobExecuting = scheduler.ExcuteInfo[plan.Job.Name]; jobExecuting {
		return
	}
	fmt.Println("构建执行态", plan.Job.Name)
	//构建执行状态
	scheduler.ExcuteInfo[plan.Job.Name] = plan.BuildJobExecuteInfo()

	//执行任务
	ExecutorG.ExecuteJob(scheduler.ExcuteInfo[plan.Job.Name])
}

//检查所有到期任务并执行
func InitScheduler() (err error) {

	Schedule = &Scheduler{
		EventChan:    make(chan *common.Event, 1000),
		SchedulePlan: make(map[string]*common.JobSchedulePlan),
		ExcuteInfo:   make(map[string]*common.JobExecuteInfo),
		ResultChan:   make(chan *common.JobExecuteResult, 1000),
	}

	go Schedule.Loop()
	return
}

func (s *Scheduler) Loop() {
	var (
		jobEvent      *common.Event
		scheduleNext  time.Duration
		scheduleTimer *time.Timer
		result        *common.JobExecuteResult
	)
	//初始化 获取下次调度时间
	scheduleNext = Schedule.TrySchedule()

	//调度器的延迟定时器
	scheduleTimer = time.NewTimer(scheduleNext)

	for {
		select {
		case <-scheduleTimer.C:
			//监听到了任务变更
		case jobEvent = <-s.EventChan:
			//对任务进行增删改查
			s.HandleJobEvent(jobEvent)
		case result = <-s.ResultChan:
			//对任务结果进行处理
			s.HandleJobResult(result)
		}
		//执行到期任务并返回下次最近任务执行时间
		scheduleNext = Schedule.TrySchedule()
		//重置调度时间间隔
		scheduleTimer.Reset(scheduleNext)
	}
}

func (s *Scheduler) PushJobEvent(event *common.Event) {
	s.EventChan <- event
}

func (s *Scheduler) HandleJobEvent(jobEvent *common.Event) {
	var (
		plan        *common.JobSchedulePlan
		execute     *common.JobExecuteInfo
		err         error
		jobExist    bool
		excuteExist bool
	)
	switch jobEvent.EventType {
	case common.EVENT_SAVE:
		if plan, err = jobEvent.Job.BuildJobSchedulePlan(); err == nil {
			Schedule.SchedulePlan[jobEvent.Job.Name] = plan
		} else {
			log.Fatal("job cron 配置错误 ,job name ", jobEvent.Job.Name)
		}
	case common.EVENT_DELETE:
		if plan, jobExist = Schedule.SchedulePlan[jobEvent.Job.Name]; jobExist {
			delete(Schedule.SchedulePlan, jobEvent.Job.Name)
		}
	case common.EVENT_KILL: //强杀任务事件
		//fmt.Printf("get kill event %#v\n",*jobEvent.Job)
		//fmt.Printf("get kill excute info %#v\n",Schedule.ExcuteInfo)

		if execute, excuteExist = Schedule.ExcuteInfo[jobEvent.Job.Name]; excuteExist {
			execute.CancelFunc()
		}
	}
}

func (s *Scheduler) PushJobResult(result *common.JobExecuteResult) {
	s.ResultChan <- result
}

//处理任务结果
func (s *Scheduler) HandleJobResult(result *common.JobExecuteResult) {
	//删除执行状态
	var (
		log *common.JobLog
	)
	delete(s.ExcuteInfo, result.ExecuteInfo.Job.Name)
	//
	if result.Err != common.ERROR_LOCK_ALREADY {
		log = &common.JobLog{
			JobName:      result.ExecuteInfo.Job.Name,
			Command:      result.ExecuteInfo.Job.Command,
			Output:       string(result.Output),
			PlanTime:     result.ExecuteInfo.PlanTime.UnixNano() / 1000 / 1000,
			ScheduleTime: result.ExecuteInfo.ActualTIme.UnixNano() / 1000 / 1000,
			StartTime:    result.StartTime.UnixNano() / 1000 / 1000,
			EndTime:      result.EndTime.UnixNano() / 1000 / 1000,
		}
		if result.Err != nil {
			log.Err = result.Err.Error()
		}
		//存储到mongodb
		LogDb.Append(log)
	}
	fmt.Printf("任务执行完成 %s,err %v\n", result.ExecuteInfo.Job.Name, result.Err)
}
