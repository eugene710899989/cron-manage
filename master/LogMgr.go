package master

import (
	"cronSystem/common"
	"fmt"
	"github.com/globalsign/mgo"
	"net/url"
	"os"
	"strconv"
	"time"
)

type LogMgr struct {
	client        *mgo.Database
	logCollection *mgo.Collection
}

type MongoConnConfig struct {
	Username   string
	Password   string
	ServerList string
	Database   string
	Option     string
}

type LogCondition struct {
	JobName string `bson:"jobName"`
}

type LogSort struct {
	SortOrder int `bson:"startTime"`
}

type LogRecord struct {
	JobName   string `bson:"jobName"`
	Command   string `bson:"command"`
	Err       string `bson:"err"`
	Content   string `bson:"content"`
	TimePoint `bson:"timePoint"`
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

var (
	LogDb *LogMgr
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
	LogDb = &LogMgr{client: mg, logCollection: collection}
	return nil
}

func (j *LogMgr) LogList(name string, skip, limit int) (logArr []*common.JobLog, err error) {
	var (
		filter *LogCondition
		query  *mgo.Query
	)
	logArr = make([]*common.JobLog, 0)
	filter = &LogCondition{JobName: name}
	query = j.logCollection.Find(filter).Sort("-startTimes").Limit(limit).Skip(skip)
	err = query.All(&logArr)
	return
}
