package schedule

import (
	"fmt"
	"github.com/gorhill/cronexpr"
	"time"
)

func CronValidate(cronStr string) time.Time {
	expr, err := cronexpr.Parse(cronStr)
	if err != nil {
		panic(err)
	}
	fmt.Println(expr.Next(time.Now()))
	return expr.Next(time.Now())
}
