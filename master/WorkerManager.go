package master

import (
	"context"
	"cron-manage/common"
	"github.com/coreos/etcd/clientv3"
	"os"
	"strconv"
	"strings"
	"time"
)

type WorkManager struct {
	client *clientv3.Client
	kv     clientv3.KV
	lease  clientv3.Lease
}

var (
	WorkMrg *WorkManager
)

//初始化管理器
func InitWorkMgr() (err error) {
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
	WorkMrg = &WorkManager{kv: kv, lease: lease, client: client}
	return
}

func (jm *WorkManager) WorkList() (workers []string, err error) {
	var (
		listResp *clientv3.GetResponse
		jobKey   string
	)
	jobKey = common.WORKER_PREFIX
	//保存到etcd
	if listResp, err = jm.kv.Get(context.TODO(), jobKey, clientv3.WithPrefix()); err != nil {
		return
	}
	workers = make([]string, 0)
	for _, v := range listResp.Kvs {
		workers = append(workers, common.ExtractWorkIp(string(v.Key)))
	}
	return
}
