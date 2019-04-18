package master

import (
	"github.com/eugene710899989/cron-manage/common"
	"encoding/json"
	"flag"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"
)

type ApiServer struct {
	server *http.Server
}

//任务保存到etcd
//解析Post json
func handleJobSave(w http.ResponseWriter, r *http.Request) {
	var (
		postJob string
		err     error
		job     common.Job
		oldJob  *common.Job
		bytes   []byte
	)
	if postJob, err = common.GetPostData(r, "job"); err != nil {
		goto ERR
	}

	//取表单字段
	if err = json.Unmarshal([]byte(postJob), &job); err != nil {
		goto ERR
	}

	//保存到Etcd
	if oldJob, err = JobManager.SaveJob(&job); err != nil {
		goto ERR
	}

	//返回应答
	if bytes, err = common.HttpReturn(0, "success", oldJob); err != nil {
		goto ERR
	}
	w.Write(bytes)
	return
ERR:
	if bytes, err = common.HttpReturn(-1, err.Error(), nil); err != nil {
		w.Write(bytes)
	} else {
		panic(err)
	}
}

func handleJobDelete(w http.ResponseWriter, r *http.Request) {
	var (
		jobName string
		err     error
		oldJob  *common.Job
		bytes   []byte
	)
	if jobName, err = common.GetPostData(r, "name"); err != nil {
		goto ERR
	}

	//保存到Etcd
	if oldJob, err = JobManager.DeleteJob(jobName); err != nil {
		goto ERR
	}

	//返回应答
	if bytes, err = common.HttpReturn(0, "success", oldJob); err != nil {
		goto ERR
	}
	w.Write(bytes)
	return
ERR:
	if bytes, err = common.HttpReturn(-1, err.Error(), nil); err != nil {
		w.Write(bytes)
	} else {
		panic(err)
	}
}

func handleJobList(w http.ResponseWriter, r *http.Request) {
	var (
		err   error
		jobs  []*common.Job
		bytes []byte
	)

	//保存到Etcd
	if jobs, err = JobManager.JobList(); err != nil {
		goto ERR
	}

	//返回应答
	if bytes, err = common.HttpReturn(0, "success", jobs); err != nil {
		goto ERR
	}
	w.Write(bytes)
	return
ERR:
	if bytes, err = common.HttpReturn(-1, err.Error(), nil); err != nil {
		w.Write(bytes)
	}
}

//查询任务日志
func handleJobLog(w http.ResponseWriter, r *http.Request) {
	var (
		err        error
		logs       []*common.JobLog
		bytes      []byte
		jobName    string
		skipParam  string
		limitParam string
		skip       int
		limit      int
		request    url.Values
	)
	request, err = common.GetRequestData(r)
	if err != nil {
		goto ERR
	}
	skipParam = request.Get("skip")
	limitParam = request.Get("limit")
	jobName = request.Get("name")
	if skip, err = strconv.Atoi(skipParam); err != nil {
		skip = 0
	}
	if limit, err = strconv.Atoi(limitParam); err != nil {
		limit = 10
	}

	if logs, err = LogDb.LogList(jobName, skip, limit); err != nil {
		goto ERR
	}
	//返回应答
	if bytes, err = common.HttpReturn(0, "success", logs); err != nil {
		goto ERR
	}
	w.Write(bytes)
	return
ERR:
	if bytes, err = common.HttpReturn(-1, err.Error(), nil); err != nil {
		w.Write(bytes)
	}
}

func handleJobKiller(w http.ResponseWriter, r *http.Request) {
	var (
		err     error
		bytes   []byte
		jobName string
	)

	if jobName, err = common.GetPostData(r, "name"); err != nil {
		return
	}

	//保存到Etcd
	if err = JobManager.JobStop(jobName); err != nil {
		goto ERR
	}

	//返回应答
	if bytes, err = common.HttpReturn(0, "success", nil); err != nil {
		goto ERR
	}
	w.Write(bytes)
	return
ERR:
	if bytes, err = common.HttpReturn(-1, err.Error(), nil); err != nil {
		w.Write(bytes)
	}
}

var (
	//单例对象
	Server *ApiServer
	Args   map[string]string
)

//初始化命令行参数
func InitArgs() (err error) {
	var (
		configFile string
	)
	Args = make(map[string]string)
	flag.Parse()
	flag.StringVar(&configFile, "config", "", "config file path")
	Args["configPath"] = configFile
	return
}

func InitApiServer() (err error) {
	//初始化

	var (
		mux                       *http.ServeMux
		listener                  net.Listener
		server                    *http.Server
		readTimeOut, writeTimeOut int
		staticDir                 http.Dir
		staticHandler             http.Handler //静态文件管理器
	)

	mux = http.NewServeMux()
	mux.HandleFunc("/job/save", handleJobSave)
	mux.HandleFunc("/job/delete", handleJobDelete)
	mux.HandleFunc("/job/list", handleJobList)
	mux.HandleFunc("/job/kill", handleJobKiller)
	mux.HandleFunc("/job/log", handleJobLog)
	mux.HandleFunc("/worker/list", handleWorkerList)

	//页面文件
	//staticDir = http.Dir(os.Getenv("webroot"))
	staticDir = http.Dir(os.Getenv("webroot"))
	staticHandler = http.FileServer(staticDir)

	mux.Handle("/", http.StripPrefix("/", staticHandler))
	//mux.Handle("/", staticHandler)

	//v := reflect.ValueOf(mux).Elem()
	//fmt.Printf("routes: %#v\n", v)
	//fmt.Printf("routes: %v\n", v.FieldByName("m"))

	//listener, err = net.Listen("tcp", ":8070")
	listener, err = net.Listen("tcp", ":"+os.Getenv("apiPort"))
	if err != nil {
		return
	}
	//创建http服务
	if readTimeOut, err = strconv.Atoi(os.Getenv("apiReadTimeOut")); err != nil {
		return
	}
	if writeTimeOut, err = strconv.Atoi(os.Getenv("apiWriteTimeOut")); err != nil {
		return
	}
	server = &http.Server{
		ReadTimeout:  time.Duration(readTimeOut) * time.Millisecond,
		WriteTimeout: time.Duration(writeTimeOut) * time.Millisecond,
		Handler:      mux,
	}
	Server = &ApiServer{server: server}

	err = server.Serve(listener)
	if err != nil {
		panic(err)
	}
	return
}

func handleWorkerList(w http.ResponseWriter, r *http.Request) {
	var (
		err     error
		workers []string
		bytes   []byte
	)

	//保存到Etcd
	if workers, err = WorkMrg.WorkList(); err != nil {
		goto ERR
	}

	//返回应答
	if bytes, err = common.HttpReturn(0, "success", workers); err != nil {
		goto ERR
	}
	w.Write(bytes)
	return
ERR:
	if bytes, err = common.HttpReturn(-1, err.Error(), nil); err != nil {
		w.Write(bytes)
	}
}
