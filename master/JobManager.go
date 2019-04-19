package master

import (
	"context"
	"cron-manage/common"
	"encoding/json"
	"github.com/coreos/etcd/clientv3"
	"os"
	"strconv"
	"strings"
	"time"
)

type JobMgr struct {
	client *clientv3.Client
	kv     clientv3.KV
	lease  clientv3.Lease
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
	JobManager = &JobMgr{kv: kv, lease: lease, client: client}
	return
}

func (jm *JobMgr) SaveJob(job *common.Job) (oldJob *common.Job, err error) {
	//存在/cron/jobs/任务名->json
	var (
		putResp  *clientv3.PutResponse
		jobKey   string
		jobValue []byte
	)
	jobKey = common.JOB_PREFIX + job.Name
	if jobValue, err = json.Marshal(job); err != nil {
		return
	}
	//保存到etcd
	if putResp, err = jm.kv.Put(context.TODO(), jobKey, string(jobValue), clientv3.WithPrevKV()); err != nil {
		return
	}
	//判断如果是更新，返回旧值
	if putResp.PrevKv != nil {
		json.Unmarshal(putResp.PrevKv.Value, &oldJob)
	}
	return
}

//删除任务
func (jm *JobMgr) DeleteJob(key string) (job *common.Job, err error) {

	var (
		oldJob  common.Job
		delResp *clientv3.DeleteResponse
		jobKey  string
	)
	jobKey = common.JOB_PREFIX + key
	//保存到etcd
	if delResp, err = jm.kv.Delete(context.TODO(), jobKey, clientv3.WithPrevKV()); err != nil {
		return
	}
	if len(delResp.PrevKvs) > 0 {
		json.Unmarshal(delResp.PrevKvs[0].Value, &oldJob)
		job = &oldJob
	}
	return
}

func (jm *JobMgr) JobList() (jobs []*common.Job, err error) {
	var (
		listResp *clientv3.GetResponse
		jobKey   string
	)
	jobKey = common.JOB_PREFIX
	//保存到etcd
	if listResp, err = jm.kv.Get(context.TODO(), jobKey, clientv3.WithPrefix()); err != nil {
		return
	}
	jobs = make([]*common.Job, 0)
	for _, v := range listResp.Kvs {
		var job common.Job
		err = json.Unmarshal(v.Value, &job)
		if err != nil {
			continue
		}
		jobs = append(jobs, &job)
	}
	return
}

func (jm *JobMgr) JobStop(name string) (err error) {
	//更新 /cron/killers/任务名
	var (
		jobKey         string
		leaseGrandResp *clientv3.LeaseGrantResponse
		leaseId        clientv3.LeaseID
	)

	jobKey = common.KILLER_PREFIX + name
	//保存到etcd 通知killer杀死对应任务 worker监听killer目录下put操作
	//带过期时间。即使杀死任务失败也会自动过期 1s后
	leaseGrandResp, err = jm.lease.Grant(context.TODO(), 1)
	if err != nil {
		return
	}
	leaseId = leaseGrandResp.ID
	if _, err = jm.kv.Put(context.TODO(), jobKey, "", clientv3.WithLease(leaseId)); err != nil {
		return
	}

	return

}

func (jm *JobMgr) JobLog(name string) (logs []*common.JobLog, err error) {
	//更新 /cron/killers/任务名
	var (
	//jobKey         string
	)
	//condition := LogRecord{JobName: "job1"}
	//query := collection.Find(condition).Limit(2).Skip(0)
	//re := []LogRecord{}
	//err = query.All(&re)
	//if err != nil {
	//	panic(err)
	//	return
	//}
	//jobKey = common.KILLER_PREFIX + name

	return

}

//etcd --name public_infra --initial-advertise-peer-urls http://10.0.22.38:2380 \
//--listen-peer-urls http://10.0.22.38:2380 \
//--listen-client-urls http://10.0.22.38:2379 \
//--advertise-client-urls http://10.0.22.38:2379 \
//--initial-cluster-token eugene-cluster \
//--initial-cluster public_infra=http://10.0.22.38:2380,public_infra1=http://10.0.22.38:12380,public_infra2=http://10.0.22.38:22380 \
//--initial-cluster-state new
//
//
//etcd --name public_infra1 --initial-advertise-peer-urls http://10.0.22.38:12380 \
//--listen-peer-urls http://10.0.22.38:12380 \
//--listen-client-urls http://10.0.22.38:12379 \
//--advertise-client-urls http://10.0.22.38:12379 \
//--initial-cluster-token eugene-cluster \
//--initial-cluster public_infra=http://10.0.22.38:2380,public_infra1=http://10.0.22.38:12380,public_infra2=http://10.0.22.38:22380 \
//--initial-cluster-state new
//
//etcd --name public_infra2 --initial-advertise-peer-urls http://10.0.22.38:22380 \
//--listen-peer-urls http://10.0.22.38:22380 \
//--listen-client-urls http://10.0.22.38:22379 \
//--advertise-client-urls http://10.0.22.38:22379 \
//--initial-cluster-token eugene-cluster \
//--initial-cluster public_infra=http://10.0.22.38:2380,public_infra1=http://10.0.22.38:12380,public_infra2=http://10.0.22.38:22380 \
//--initial-cluster-state new

//
//etcd1: etcd --name infra1 --listen-client-urls http://10.0.22.38:2379 --advertise-client-urls http://10.0.22.38:2379 --listen-peer-urls http://10.0.22.38:12380 --initial-advertise-peer-urls http://10.0.22.38:12380 --initial-cluster-token etcd-cluster-1 --initial-cluster 'infra1=http://10.0.22.38:12380,infra2=http://10.0.22.38:22380,infra3=http://10.0.22.38:32380' --initial-cluster-state new --enable-pprof
//etcd2: etcd --name infra2 --listen-client-urls http://10.0.22.38:22379 --advertise-client-urls http://10.0.22.38:22379 --listen-peer-urls http://10.0.22.38:22380 --initial-advertise-peer-urls http://10.0.22.38:22380 --initial-cluster-token etcd-cluster-1 --initial-cluster 'infra1=http://10.0.22.38:12380,infra2=http://10.0.22.38:22380,infra3=http://10.0.22.38:32380' --initial-cluster-state new --enable-pprof
//etcd3: etcd --name infra3 --listen-client-urls http://10.0.22.38:32379 --advertise-client-urls http://10.0.22.38:32379 --listen-peer-urls http://10.0.22.38:32380 --initial-advertise-peer-urls http://10.0.22.38:32380 --initial-cluster-token etcd-cluster-1 --initial-cluster 'infra1=http://10.0.22.38:12380,infra2=http://10.0.22.38:22380,infra3=http://10.0.22.38:32380' --initial-cluster-state new --enable-pprof