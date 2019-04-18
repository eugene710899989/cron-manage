package worker

import (
	"context"
	"cron-manage/common"
	"fmt"
	"github.com/coreos/etcd/clientv3"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

//注册节点到etcd
type RegisterM struct {
	client  *clientv3.Client
	kv      clientv3.KV
	lease   clientv3.Lease
	localIp string
}

var (
	Register *RegisterM
)

func InitRegister() (err error) {
	var (
		config       clientv3.Config
		client       *clientv3.Client
		kv           clientv3.KV
		lease        clientv3.Lease
		etcdTimeout  int
		etcdEndPoint []string
		ip           string
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
	ip, err = getLocalIp()
	if err != nil {
		return
	}
	//赋值给单例
	fmt.Println("register service")
	Register = &RegisterM{kv: kv, lease: lease, client: client, localIp: ip}
	go Register.Register()
	return nil
}

func getLocalIp() (ipv4 string, err error) {
	var (
		addrs []net.Addr
		addr  net.Addr
		IPNet *net.IPNet
		isIp  bool
	)
	//获取所有网卡地址
	if addrs, err = net.InterfaceAddrs(); err != nil {
		return
	}
	//获取第一个非 localhost的地址
	for _, addr = range addrs {
		//是ip地址且不是环回地址
		if IPNet, isIp = addr.(*net.IPNet); isIp && !IPNet.IP.IsLoopback() {
			//转化为ip4地址不为空说明是ipv4地址
			if IPNet.IP.To4() != nil {
				ipv4 = IPNet.IP.String()
				return
			}
		}
	}

	err = common.ERROR_NO_LOCAL_IP_FOUND
	return
}

//注册到etcd /cron/worker/IP 并启动自动续租
func (Register *RegisterM) Register() (err error) {
	var (
		leaseResp        *clientv3.LeaseGrantResponse
		leaseId          clientv3.LeaseID
		keepLiveRespChan <-chan *clientv3.LeaseKeepAliveResponse
		//putResp          *clientv3.PutResponse
		registerKey   string
		CancelContext context.Context
		CancelFunc    context.CancelFunc
	)

	for {
		CancelFunc = nil
		registerKey = common.WORKER_PREFIX + Register.localIp
		if leaseResp, err = Register.lease.Grant(context.TODO(), 10); err != nil {
			goto RETRY
		}
		leaseId = leaseResp.ID
		if keepLiveRespChan, err = Register.lease.KeepAlive(context.TODO(), leaseId); err != nil {
			goto RETRY
		}
		CancelContext, CancelFunc = context.WithCancel(context.TODO())
		//put 异常取消租约
		if _, err = Register.kv.Put(CancelContext, registerKey, "", clientv3.WithLease(leaseId)); err != nil {
			goto RETRY
		}
		//处理续租应答的协程
		for {
			var (
				keepLiveResp *clientv3.LeaseKeepAliveResponse
			)
			select {
			case keepLiveResp = <-keepLiveRespChan:
				if keepLiveResp == nil {
					goto RETRY
				}
			}
		}

	}
RETRY:
	time.Sleep(time.Second)
	if CancelFunc == nil {
		CancelFunc()
	}
	return nil
}
