package main

import (
	"fmt"
	"github.com/globalsign/mgo"
	"net/url"
	"time"
)

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

func main() {
	config := MongoConnConfig{Database: "my_collection", ServerList: "127.0.0.1:27017"}
	mg, err := config.Init()
	if err != nil {
		panic(err)
	}
	collection := mg.C("log")
	condition := LogRecord{JobName: "job1"}
	query := collection.Find(condition).Limit(2).Skip(0)
	re := []LogRecord{}
	err = query.All(&re)
	if err != nil {
		panic(err)
		return
	}
	fmt.Printf("%#v\n", re)
	//collection.(condition)
	err = collection.Insert(LogRecord{JobName: "hhh",
		Command:   "fds",
		Err:       "",
		Content:   "ssss",
		TimePoint: TimePoint{time.Now().Unix(), time.Now().Unix() + 100}})
	if err != nil {
		panic(err)
	}
	delCond := &DeleteCond{beforeCond: TimeBeforeCond{Before: time.Now().Unix()}}
	info, err := collection.RemoveAll(delCond)
	if err != nil {
		panic(err)
		return
	}
	fmt.Printf("%#v\n", info)
	fmt.Printf("%#v", info.Removed)
	return
}

func (config *MongoConnConfig) Init() (mgDb *mgo.Database, err error) {
	session, err := mgo.Dial(config.GetDsn())
	if err != nil {
		return nil, err
	}
	if err = session.Ping(); err != nil {
		return nil, err
	}

	mg := session.DB(config.Database)
	return mg, nil
}
