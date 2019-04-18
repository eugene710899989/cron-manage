package main

import (
	"github.com/eugene710899989/cron-manage/worker"
	"fmt"
	"github.com/joho/godotenv"
	"os"
	"runtime"
	"time"
)

func main() {
	var (
		err error
	)
	//充分利用Cpu核数
	runtime.GOMAXPROCS(runtime.NumCPU())
	root, err := os.Getwd()
	if err != nil {
		goto ERR
	}
	err = godotenv.Load(root + "/../.env")
	if err != nil {
		goto ERR
	}

	//启动执行器
	if err = worker.InitExecutor(); err != nil {
		goto ERR
	}

	//启动日志协程
	if err = worker.LogInit(); err != nil {
		goto ERR
	}

	//启动调度器
	if err = worker.InitScheduler(); err != nil {
		goto ERR
	}

	//启动任务管理器
	if err = worker.InitJobMgr(); err != nil {
		goto ERR
	}

	//注册服务
	if err = worker.InitRegister(); err != nil {
		goto ERR
	}

	for {
		time.Sleep(time.Second)
	}
	return
ERR:
	fmt.Println(err)
}
