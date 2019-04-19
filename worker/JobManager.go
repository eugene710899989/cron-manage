package worker

import (
	"context"
	"cron-manage/common"
	"github.com/coreos/etcd/mvcc/mvccpb"
	"go.etcd.io/etcd/clientv3"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

type JobMgr struct {
	client  *clientv3.Client
	kv      clientv3.KV
	lease   clientv3.Lease
	watcher clientv3.Watcher
}

var (
	JobManager *JobMgr
)

//初始化管理器
func InitJobMgr() (err error) {
	var (
		config       clientv3.Config
		client       *clientv3.Client
		kv           clientv3.KV
		lease        clientv3.Lease
		etcdTimeout  int
		etcdEndPoint []string
	)
	if etcdTimeout, err = strconv.Atoi(os.Getenv("etcdTimeOut")); err != nil {
		return
	}
	if etcdEndPoint = strings.Split(os.Getenv("etcdEndPoints"), ","); err != nil {
		return
	}
	config = clientv3.Config{
		Endpoints:   etcdEndPoint,
		DialTimeout: time.Duration(etcdTimeout) * time.Millisecond,
	}

	//建立连接
	if client, err = clientv3.New(config); err != nil {
		return
	}
	kv = clientv3.NewKV(client)
	lease = clientv3.NewLease(client)

	//赋值给单例
	JobManager = &JobMgr{kv: kv, lease: lease, client: client, watcher: clientv3.NewWatcher(client)}

	err = JobManager.WatchJob()
	if err != nil {
		log.Fatal(err)
	}

	err = JobManager.WatchKillJob()
	if err != nil {
		log.Fatal(err)
	}
	return
}

//获取任务状态
func (j *JobMgr) WatchJob() error {
	var (
		getRe              *clientv3.GetResponse
		watchResChan       clientv3.WatchChan
		watchRes           clientv3.WatchResponse
		watchStartRevision int64
		job                *common.Job
		err                error
		jobName            string
		event              *common.Event
	)
	//getRe.Header.Revision
	if getRe, err = j.kv.Get(context.TODO(), common.JOB_PREFIX, clientv3.WithPrefix()); err != nil {
		return err
	}
	for _, kvpair := range getRe.Kvs {
		job = &common.Job{}
		if err = common.UnPackJob(kvpair.Value, job); err == nil {
			event = common.BuildEvent(common.EVENT_SAVE, job)
			//把job同步给schedule协程处理
			Schedule.PushJobEvent(event)
		} else {
			log.Println(err)
		}

	}

	go func() {

		//当前etcd 集群事务id 单调递增，因此监听这个版本+1的变化
		watchStartRevision = getRe.Header.Revision + 1
		watchResChan = j.watcher.Watch(context.TODO(), common.JOB_PREFIX, clientv3.WithRev(watchStartRevision), clientv3.WithPrefix())
		for watchRes = range watchResChan {
			for _, resp := range watchRes.Events {
				switch resp.Type {
				case mvccpb.PUT:
					job = &common.Job{}
					if err = common.UnPackJob(resp.Kv.Value, job); err != nil {
						continue
					}

					event = common.BuildEvent(common.EVENT_SAVE, job)
				case mvccpb.DELETE:
					jobName = common.ExtractJobName(string(resp.Kv.Key))
					event = common.BuildEvent(common.EVENT_DELETE, &common.Job{Name: jobName})
				}
				// 反序列化后推送创建事件给schedule协程
				// 推删除事件给调度协程
				Schedule.PushJobEvent(event)
			}
		}
	}()

	return nil
}

//获取强杀任务状态
func (j *JobMgr) WatchKillJob() error {
	var (
		watchResChan clientv3.WatchChan
		watchRes     clientv3.WatchResponse
		job          *common.Job
		jobName      string
		event        *common.Event
	)

	go func() {

		watchResChan = j.watcher.Watch(context.TODO(), common.KILLER_PREFIX, clientv3.WithPrefix())
		for watchRes = range watchResChan {
			for _, resp := range watchRes.Events {
				switch resp.Type {
				case mvccpb.PUT: //获取杀死任务事件
					jobName = common.ExtractKillerName(string(resp.Kv.Key))
					job = &common.Job{Name: jobName}
					event = common.BuildEvent(common.EVENT_KILL, job)
					Schedule.PushJobEvent(event)
				case mvccpb.DELETE: //killer 标记过期被自动删除
				}
				// 反序列化后推送创建事件给schedule协程
				// 推删除事件给调度协程
			}
		}
	}()

	return nil
}

//创建任务执行锁
func (j *JobMgr) CreateJobLock(jobName string) *Locker {
	//返回一把锁
	return InitJobLock(jobName, j.kv, j.lease)
}
