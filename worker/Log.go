package worker

import (
	"github.com/eugene710899989/cron-manage/common"
	"fmt"
	"github.com/globalsign/mgo"
	"net/url"
	"os"
	"strconv"
	"time"
)

type LogSink struct {
	client        *mgo.Database
	logCollection *mgo.Collection
	logChan       chan *common.JobLog
	commitChan    chan *LogBatch
}

type MongoConnConfig struct {
	Username   string
	Password   string
	ServerList string
	Database   string
	Option     string
}

type LogRecord struct {
	JobName   string `bson:"jobName"`
	Command   string `bson:"command"`
	Err       string `bson:"err"`
	Content   string `bson:"content"`
	TimePoint `bson:"timePoint"`
}

type LogBatch struct {
	Logs []interface{}
}

type TimePoint struct {
	StartTime int64 `bson:"startTime"`
	EndTime   int64 `bson:"endTime"`
}

func (dsn *MongoConnConfig) GetDsn() string {
	if dsn.Username != "" && dsn.Password != "" {
		return fmt.Sprintf("mongodb://%s:%s@%s/%s?%s",
			url.QueryEscape(dsn.Username),
			url.QueryEscape(dsn.Password),
			dsn.ServerList,
			dsn.Database,
			url.QueryEscape(dsn.Option))
	}
	return fmt.Sprintf("mongodb://%s/%s?%s", dsn.ServerList, dsn.Database, url.QueryEscape(dsn.Option))
}

type TimeBeforeCond struct {
	Before int64 `bson:"$lt"`
}

type DeleteCond struct {
	beforeCond TimeBeforeCond `bson:"timePoint.startTime"`
}

var (
	LogDb *LogSink
)

func LogInit() (err error) {
	var (
		timeOut int
	)
	config := MongoConnConfig{Database: os.Getenv("mongoDbName"), ServerList: os.Getenv("mongoAddress")}
	//设置超时时间为5s

	if timeOut, err = strconv.Atoi(os.Getenv("mongoTimeOut")); err != nil {
		timeOut = 5000
	}
	session, err := mgo.DialWithTimeout(config.GetDsn(), time.Millisecond*time.Duration(timeOut))
	if err != nil {
		return err
	}
	if err = session.Ping(); err != nil {
		return err
	}

	mg := session.DB(config.Database)
	if err != nil {
		panic(err)
	}
	collection := mg.C("log")
	LogDb = &LogSink{client: mg, logCollection: collection, logChan: make(chan *common.JobLog, 1000), commitChan: make(chan *LogBatch, 100)}
	go LogDb.LoopSave()
	return nil
}

func (LogSink *LogSink) LoopSave() {
	var (
		log          *common.JobLog
		err          error
		batch        *LogBatch
		timeoutBatch *LogBatch
		size         int
		wait         int
		logInterval  *time.Timer
	)
	if wait, err = strconv.Atoi(os.Getenv("logInterval")); err != nil {
		panic(err)
	}
	if size, err = strconv.Atoi(os.Getenv("logBatchSize")); err != nil || size == 0 {
		size = 5000
	}
	for {
		select {

		case log = <-LogSink.logChan:
			if batch == nil {
				batch = &LogBatch{}
				logInterval = time.AfterFunc(time.Duration(wait)*time.Millisecond, func(batch *LogBatch) func() {
					//发出超时通知，不直接提交batch 以免并发提交
					return func() {
						LogSink.commitChan <- batch
					}
				}(batch))
			}
			batch.Logs = append(batch.Logs, log)
			//如果批次满了或到时间间隔了，立即保存
			if len(batch.Logs) >= size {
				err = LogSink.logCollection.Insert(batch.Logs...)
				batch = nil
				logInterval.Stop()
			}
			//保存超时批次
		case timeoutBatch = <-LogSink.commitChan:

			//判断过期批次是否是未打满批次以免同时打满和到期并发插入
			if timeoutBatch != batch {
				fmt.Println("not same batch")
				continue
			}
			err = LogSink.logCollection.Insert(batch.Logs...)
			batch = nil //重新启动定时器
		}
	}

}

func (sink *LogSink) Append(log *common.JobLog) {
	select {
	//应对chan 满了的情况不阻塞
	case sink.logChan <- log:
	default:

	}
}
