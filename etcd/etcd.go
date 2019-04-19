package main

import (
	"context"
	"fmt"
	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/mvcc/mvccpb"
	"time"
)

func watchDemo() {
	var (
		getRe              *clientv3.GetResponse
		watchStartRevision int64
		watcher            clientv3.Watcher
		watchResChan       clientv3.WatchChan
		watchResp          clientv3.WatchResponse
	)
	config := clientv3.Config{Endpoints: []string{"127.0.0.1:2379"},
		DialTimeout: 5 * time.Second,
	}
	//建立客户端
	client, err := clientv3.New(config)
	if err != nil {
		panic(err)
	}
	//用于读取client键值对
	kv := clientv3.NewKV(client)
	go func() {
		for {
			kv.Put(context.TODO(), "/job/tasks/j4", "hsdf")
			kv.Delete(context.TODO(), "/job/tasks/j4")
			time.Sleep(time.Second)
		}
	}()

	//get到当前值，并监听后续变化
	if getRe, _ = kv.Get(context.TODO(), "/job/tasks/j4"); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(getRe)

	//确定现在key是存在的
	if len(getRe.Kvs) > 0 {
		fmt.Printf("当前值%s\n", string(getRe.Kvs[0].Value))
	} else {
		fmt.Println("当前无值")
	}

	//当前etcd 集群事务id 单调递增，因此监听这个版本+1的变化
	//getRe.Header.Revision
	watchStartRevision = getRe.Header.Revision + 1
	fmt.Printf("current revision %d\n", watchStartRevision)
	//kv.Put(context.TODO(), "/job/tasks/jjjj", "www")
	//getRe, _ = kv.Get(context.TODO(), "/job/tasks/jjjj")
	//fmt.Printf("another revision %d\n", getRe.Header.Revision)

	watcher = clientv3.NewWatcher(client)

	ctx, cancel := context.WithCancel(context.TODO())

	time.AfterFunc(5*time.Second, func() {
		cancel()
	})
	//以集群事务id大小 监听
	//watchResChan = watcher.Watch(context.TODO(), "/job/tasks/j4", clientv3.WithRev(watchStartRevision))
	//已prefix 监听
	watchResChan = watcher.Watch(ctx, "/job/tasks", clientv3.WithPrefix())
	fmt.Printf("开始监听 start version%d\n", watchStartRevision)

	for watchResp = range watchResChan {
		for _, resp := range watchResp.Events {
			switch resp.Type {
			case mvccpb.PUT:
				fmt.Printf("watcher resp resp 修改为 %#v revision %d ,%d\n", string(resp.Kv.Value), resp.Kv.CreateRevision, resp.Kv.ModRevision)
			case mvccpb.DELETE:
				fmt.Printf("watcher resp resp delete detect %#v,modrevision %d\n", resp, resp.Kv.ModRevision)

			}
			fmt.Printf("watcher resp resp %#v\n", resp)
		}
	}

}

func main() {
	//包含 增删改和续租相关例子
	//simpleDemo()

	//watchDemo()
	//opDemo()
	TxDemo()
}

func simpleDemo() {
	var (
		getRe      *clientv3.GetResponse
		delRe      *clientv3.DeleteResponse
		keepLiveRe <-chan *clientv3.LeaseKeepAliveResponse
	)
	config := clientv3.Config{Endpoints: []string{"127.0.0.1:2379"},
		DialTimeout: 5 * time.Second,
	}
	//建立客户端
	client, err := clientv3.New(config)
	if err != nil {
		panic(err)
	}
	//用于读取client键值对
	kv := clientv3.NewKV(client)
	//option 获取put 上一次的kv
	putre, err := kv.Put(context.TODO(), "/job/tasks/j1", "hhhh", clientv3.WithPrevKV())
	putre, err = kv.Put(context.TODO(), "/job/tasks/j2", "sdfs", clientv3.WithPrevKV())
	//带超时的上下文 5秒后取消自动续租
	ctxWithTimeout, _ := context.WithTimeout(context.TODO(), 15*time.Second)
	//申请租约 以秒为单位 3s后自动过期
	lease := clientv3.NewLease(client)
	leaseRe, err := lease.Grant(context.TODO(), 3)
	if err != nil {
		panic(err)
	}
	leaseId := leaseRe.ID
	//续租 自动和手动一次的
	keepLiveRe, _ = lease.KeepAlive(ctxWithTimeout, leaseId)
	//keepLiveRe, _ = lease.KeepAlive(context.TODO(), leaseId)
	putWithLeaseRe, _ := kv.Put(context.TODO(), "/job/tasks/j3", "vvv", clientv3.WithLease(leaseId))
	fmt.Println(putWithLeaseRe)
	//处理续租应答的协程
	go func() {
		for {
			select {
			case v := <-keepLiveRe:
				if v != nil {
					fmt.Println("keep alive resp :", v)
				} else {
					goto END
				}
			}
		}
	END:
		return
	}()
	for {
		getRe, err = kv.Get(context.TODO(), "/job/tasks/j3")
		if err != nil {
			fmt.Println(err)
			break
		} else {
			fmt.Println(getRe)
			if getRe.Count == 0 {
				fmt.Println("kv 过期")

				break
			}
		}
		select {
		case <-time.Tick(1 * time.Second):
		}
	}
	time.Sleep(time.Second * 50)
	delRe, err = kv.Delete(context.TODO(), "/job/tasks/j2", clientv3.WithPrevKV())
	if len(delRe.PrevKvs) > 0 {
		fmt.Printf("%#v", delRe)
	}
	//读取prefix "/job/tasks/"的所有kv
	if getRe, err := kv.Get(context.TODO(), "/job/tasks/", clientv3.WithPrefix()); err != nil {
		panic(err)
	} else {
		//获取成功后 遍历所有kv
		fmt.Println(getRe.Kvs)
	}
	if err != nil {
		panic(err)
	}
	//fmt.Printf("%#v\n", putre)
	fmt.Printf("%#v\n", getRe)
	//fmt.Printf("%#v\n", putre.Header.Revision)
	if putre.PrevKv != nil {

		fmt.Printf("%#v\n", string(putre.PrevKv.Value))
		fmt.Printf("%#v\n", string(putre.PrevKv.Key))
	}
	//fmt.Println(re.revision)
	defer client.Close()
}

func opDemo() {
	var (
		putOp, getOp            clientv3.Op
		OpResponse, GOpResponse clientv3.OpResponse
		putOpResponse           *clientv3.PutResponse
		getOpResponse           *clientv3.GetResponse
	)
	config := clientv3.Config{Endpoints: []string{"127.0.0.1:2379"},
		DialTimeout: 5 * time.Second,
	}
	//建立客户端
	client, err := clientv3.New(config)
	if err != nil {
		panic(err)
	}
	kv := clientv3.NewKV(client)

	//获取op
	putOp = clientv3.OpPut("/test", "123")

	//执行op
	OpResponse, err = kv.Do(context.TODO(), putOp)
	if err != nil {
		panic(err)
	}
	putOpResponse = OpResponse.Put()
	fmt.Println(putOpResponse.Header)

	getOp = clientv3.OpGet("/test")
	GOpResponse, err = kv.Do(context.TODO(), getOp)
	getOpResponse = GOpResponse.Get()
	fmt.Printf("%#v", getOpResponse.Kvs[0].Value)
	defer client.Close()
}

func TxDemo() {
	//lease 实现锁自动过期
	// op操作
	// txn事务

	//上锁 创建租约强占key

	var (
		getRe         *clientv3.GetResponse
		delRe         *clientv3.DeleteResponse
		keepLiveRe    <-chan *clientv3.LeaseKeepAliveResponse
		cancelFunc    context.CancelFunc
		ctxWithCancel context.Context
		txn           clientv3.Txn
	)
	config := clientv3.Config{Endpoints: []string{"127.0.0.1:2379"},
		DialTimeout: 5 * time.Second,
	}
	//建立客户端
	client, err := clientv3.New(config)
	if err != nil {
		panic(err)
	}
	//用于读取client键值对
	kv := clientv3.NewKV(client)
	//option 获取put 上一次的kv
	putre, err := kv.Put(context.TODO(), "/job/tasks/j1", "hhhh", clientv3.WithPrevKV())
	putre, err = kv.Put(context.TODO(), "/job/tasks/j2", "sdfs", clientv3.WithPrevKV())
	//带取消的上下文 5秒后取消自动续租
	//ctxWithCancel, _ := context.WithTimeout(context.TODO(), 15*time.Second)
	ctxWithCancel, cancelFunc = context.WithCancel(context.TODO())
	//申请租约 以秒为单位 3s后自动过期
	lease := clientv3.NewLease(client)
	leaseRe, err := lease.Grant(context.TODO(), 3)
	if err != nil {
		panic(err)
	}
	leaseId := leaseRe.ID
	//续租 自动和手动一次的
	keepLiveRe, _ = lease.KeepAlive(ctxWithCancel, leaseId)
	//keepLiveRe, _ = lease.KeepAlive(context.TODO(), leaseId)
	putWithLeaseRe, _ := kv.Put(context.TODO(), "/job/tasks/j3", "vvv", clientv3.WithLease(leaseId))
	fmt.Println(putWithLeaseRe)
	//处理续租应答的协程
	go func() {
		for {
			select {
			case v := <-keepLiveRe:
				if v != nil {
					fmt.Println("keep alive resp :", v)
				} else {
					goto END
				}
			}
		}
	END:
		return
	}()

	//创建事务
	txn = kv.Txn(context.TODO())
	//test的create revison = 0
	//强占key if 不存在key 设置他 ，否则抢锁失败

	txn.If(clientv3.Compare(clientv3.CreateRevision("test"), "=", 0)).Then(clientv3.OpPut("test", "333", clientv3.WithLease(leaseId))).Else(clientv3.OpGet("test"))
	txnResp, err := txn.Commit()

	//判断是否抢到
	//txnResp.Responses 内放执行的op的结果
	if !txnResp.Succeeded {
		fmt.Println("锁被占用", string(txnResp.Responses[0].GetResponseRange().Kvs[0].Value))
		return
	}
	fmt.Println("抢到锁成功")
	time.Sleep(5 * time.Second)
	for {
		getRe, err = kv.Get(context.TODO(), "/job/tasks/j3")
		if err != nil {
			fmt.Println(err)
			break
		} else {
			fmt.Println(getRe)
			if getRe.Count == 0 {
				fmt.Println("kv 过期")

				break
			}
		}
		select {
		case <-time.Tick(1 * time.Second):
		}
	}
	defer cancelFunc()
	defer lease.Revoke(context.TODO(), leaseId)

	time.Sleep(time.Second * 50)
	delRe, err = kv.Delete(context.TODO(), "/job/tasks/j2", clientv3.WithPrevKV())
	if len(delRe.PrevKvs) > 0 {
		fmt.Printf("%#v", delRe)
	}
	//读取prefix "/job/tasks/"的所有kv
	if getRe, err := kv.Get(context.TODO(), "/job/tasks/", clientv3.WithPrefix()); err != nil {
		panic(err)
	} else {
		//获取成功后 遍历所有kv
		fmt.Println(getRe.Kvs)
	}
	if err != nil {
		panic(err)
	}
	//fmt.Printf("%#v\n", putre)
	fmt.Printf("%#v\n", getRe)
	//fmt.Printf("%#v\n", putre.Header.Revision)
	if putre.PrevKv != nil {

		fmt.Printf("%#v\n", string(putre.PrevKv.Value))
		fmt.Printf("%#v\n", string(putre.PrevKv.Key))
	}
	defer client.Close()
	//释放锁
}
