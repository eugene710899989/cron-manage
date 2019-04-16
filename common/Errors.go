package common

import "errors"

var (
	ERROR_LOCK_ALREADY      = errors.New("锁已被占用")
	ERROR_NO_LOCAL_IP_FOUND = errors.New("没有网卡ip")
)
