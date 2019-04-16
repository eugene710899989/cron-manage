package main

import (
	"context"
	"fmt"
	"github.com/gorhill/cronexpr"
	"os/exec"
	"time"
)

type result struct {
	err    error
	output []byte
}

type CronJob struct {
	expr     *cronexpr.Expression
	nextTime time.Time
}

func main() {
	schedules := make(map[string]*CronJob)
	now := time.Now()
	var expr *cronexpr.Expression

	expr = cronexpr.MustParse(`*/5 * * * * * *`)
	cronjob := &CronJob{expr, expr.Next(now)}
	schedules["job1"] = cronjob

	expr = cronexpr.MustParse(`*/1 * * * * * *`)
	cronjob = &CronJob{expr, expr.Next(now)}
	schedules["job2"] = cronjob

	//定时检测到期任务
	go func() {
		var (
			jobName string
			job     *CronJob
			now     time.Time
		)

		for {
			now = time.Now()
			for jobName, job = range schedules {
				if job.nextTime.Before(now) || job.nextTime.Equal(now) {
					// 执行这个任务
					go func(name string) {
						fmt.Println("exec ", name)
					}(jobName)
					//计算下次调度时间
					job.nextTime = job.expr.Next(now)
				}
			}
			select {
			case <-time.NewTimer(100 * time.Millisecond).C:

			}
		}
	}()
	time.Sleep(100 * time.Second)
	//timer := time.Tick(next.Sub(time.Now()))
	//select {
	//case <-timer:
	//	fmt.Println("调度了")
	//}
	//CloseCmd()
}

//关闭某个命令
func CloseCmd() {
	var (
		cmd        *exec.Cmd
		ctx        context.Context
		cancelFunc context.CancelFunc
		res        *result
		resultChan chan *result
	)
	//结果管道
	resultChan = make(chan *result, 1000)
	ctx, cancelFunc = context.WithCancel(context.TODO())
	//关闭异步子进程
	go func() {
		cmd = exec.CommandContext(ctx, "/bin/bash", "-c", "sleep 1;sleep 30;echo hello")
		b, err := cmd.CombinedOutput()
		resultChan <- &result{err, b}
	}()
	time.Sleep(1 * time.Second)
	cancelFunc()
	res = <-resultChan
	fmt.Println(string(res.output))
	fmt.Println(res.err)
}

func getOutput() {
	var (
		cmd *exec.Cmd
	)
	cmd = exec.Command("/bin/bash", "-c", "ls -a;sleep 5;ls -l")
	//获取子进程的输出
	b, err := cmd.CombinedOutput()
	if err != nil {
		panic(err)
	}
	fmt.Println(string(b))
}

func Op() {

}
