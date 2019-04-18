package worker

import (
	"context"
	"cron-manage/common"
	"fmt"
	"go.etcd.io/etcd/clientv3"
)

type Locker struct {
	kv         clientv3.KV
	lease      clientv3.Lease
	jobName    string
	leaseId    clientv3.LeaseID
	cancelFunc context.CancelFunc
	isLocked   bool
}

//初始化锁
func InitJobLock(jobName string, kv clientv3.KV, lease clientv3.Lease) *Locker {
	var (
		locker *Locker
	)
	locker = &Locker{
		jobName: jobName,
		kv:      kv,
		lease:   lease,
	}
	return locker
}

func (locker *Locker) TryLock() (err error) {
	var (
		leaseGrantResp *clientv3.LeaseGrantResponse
		keepLiveRe     <-chan *clientv3.LeaseKeepAliveResponse
		cancelFunc     context.CancelFunc
		ctxWithCancel  context.Context
		txn            clientv3.Txn
		txnResp        *clientv3.TxnResponse
		lockerKey      string
		leaseId        clientv3.LeaseID
	)

	//创建租约
	if leaseGrantResp, err = locker.lease.Grant(context.TODO(), 3); err != nil {
		fmt.Println("抢锁失败 err", err.Error())
		return
	}

	leaseId = leaseGrantResp.ID

	//yo用于取消续租
	ctxWithCancel, cancelFunc = context.WithCancel(context.TODO())
	//申请租约 以秒为单位 3s后自动过期
	locker.cancelFunc = cancelFunc
	locker.leaseId = leaseId
	//续租 自动和手动一次的
	if keepLiveRe, err = locker.lease.KeepAlive(ctxWithCancel, leaseId); err != nil {
		goto FAIL
	}

	//处理续租应答的协程 自动续租
	go func() {
		var (
			keepRe *clientv3.LeaseKeepAliveResponse
		)

		for {
			select {
			case keepRe = <-keepLiveRe: //续租失败 可能是被取消或者其他
				if keepRe == nil {
					goto END
				}
			}
		}
	END:
		return
	}()

	//创建事务
	txn = locker.kv.Txn(context.TODO())
	//test的create revison = 0
	//强占key if 不存在key 设置他 ，否则抢锁失败
	lockerKey = common.LOCKER_PREFIX + locker.jobName

	//事务抢锁

	txn.If(clientv3.Compare(clientv3.CreateRevision(lockerKey), "=", 0)).Then(clientv3.OpPut(lockerKey, "", clientv3.WithLease(leaseId))).Else(clientv3.OpGet(lockerKey))
	if txnResp, err = txn.Commit(); err != nil {
		goto FAIL
	}

	//判断是否抢到
	//txnResp.Responses 内放执行的op的结果
	//成功返回租约id，失败释放租约

	if !txnResp.Succeeded {
		err = common.ERROR_LOCK_ALREADY
		goto FAIL
	}
	locker.isLocked = true
	fmt.Println("抢锁成功 ", lockerKey)
	return nil
FAIL:
	//释放租约
	fmt.Println("抢锁失败 ", lockerKey)
	locker.lease.Revoke(context.TODO(), locker.leaseId)
	locker.cancelFunc()
	return
}

func (locker *Locker) Unlock() {
	if locker.isLocked {
		locker.cancelFunc()
		locker.lease.Revoke(context.TODO(), locker.leaseId)
	}
}
