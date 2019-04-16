package worker

import (
	"cronSystem/common"
	"fmt"
	"os/exec"
	"time"
)

type Executor struct {
}

var (
	ExecutorG *Executor
)

//执行任务
func (executor *Executor) ExecuteJob(info *common.JobExecuteInfo) {
	go func() {
		var (
			cmd    *exec.Cmd
			err    error
			output []byte
			result *common.JobExecuteResult
			locker *Locker
		)
		result = &common.JobExecuteResult{ExecuteInfo: info, Err: err}
		result.StartTime = time.Now()

		////分布式锁强占判断是否可以执行
		locker = JobManager.CreateJobLock(info.Job.Name)
		////抢锁
		defer locker.Unlock()
		//随机睡眠时间 避免时间不一致导致的任务一直分配不均
		//time.Sleep(time.Duration(rand.Intn(500)) * time.Millisecond)
		if err = locker.TryLock(); err != nil {
			result.Err = err
			result.EndTime = time.Now()
			fmt.Println("抢锁失败")
			Schedule.PushJobResult(result)
			return
		}
		//上锁是网络请求，更新任务开始时间排除上锁时间
		result.StartTime = time.Now()

		cmd = exec.CommandContext(info.CancelContext, "/bin/bash", "-c", info.Job.Command)
		output, err = cmd.CombinedOutput()

		result.EndTime = time.Now()
		result.Output = output
		result.Err = err

		//把执行结果返回给schedule ,schedule 从executable 删掉执行完的任务
		fmt.Printf("execute job %s\n result %v", info.Job.Name, string(output))
		Schedule.PushJobResult(result)
	}()
}

//初始化执行器
func InitExecutor() error {
	ExecutorG = &Executor{}
	return nil
}
