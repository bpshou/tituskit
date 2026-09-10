package crontab

import "github.com/robfig/cron/v3"

func NewCron() *cron.Cron {
	// 精确到秒
	return cron.New(
		cron.WithSeconds(), // 设置定时为秒
		cron.WithChain(
			cron.Recover(cron.DefaultLogger),             // panic 异常处理
			cron.DelayIfStillRunning(cron.DefaultLogger), // 一个任务执行一次
		),
	)
}
