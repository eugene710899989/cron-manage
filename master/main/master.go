package main

import (
	"cron-manage/master"
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
	err = godotenv.Load(root + "/master/.env")
	if err != nil {
		goto ERR
	}

	if err = master.InitArgs(); err != nil {
		goto ERR
	}
	if err = master.LogInit(); err != nil {
		goto ERR
	}
	if err = master.InitJobMgr(); err != nil {
		goto ERR
	}
	if err = master.InitWorkMgr(); err != nil {
		goto ERR
	}

	if err = master.InitApiServer(); err != nil {
		goto ERR
	}
	for {
		time.Sleep(time.Second)
	}
	return
ERR:
	fmt.Println(err)
}
