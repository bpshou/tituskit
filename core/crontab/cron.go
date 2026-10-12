package crontab

import (
	"time"

	"github.com/robfig/cron/v3"
)

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

// NextRun 回读指定任务的下次执行时间。
//
// 时间基是调度器协调器自己的时钟，和任务实际触发用的是同一个，
// 因此在任务内部再取一次 time.Now() 得不到更准确的结果。
//
// 以下三种情况 ok 为 false：id 为 0（EntryID 的零值，合法任务从 1 开始）、
// 从未注册过的 id、该任务尚未被调度器排入下次触发时间（调度器未 Start，
// 或任务已被 Remove）。
func NextRun(c *cron.Cron, id cron.EntryID) (next time.Time, ok bool) {
	// EntryID 0 是零值，不对应任何任务。cron.Entry 对未知 id 本就会返回
	// 零值 Entry（Next 也是零值时间），所以这里并非行为修复，只是把
	// "0 不是有效 id" 这一契约显式化，避免调用方把注册失败时的返回值当有效 id。
	if id == 0 {
		return time.Time{}, false
	}

	// cron.Entry 内部加锁读取，未注册时返回零值 Entry。
	entry := c.Entry(id)
	if entry.ID != id || entry.Next.IsZero() {
		return time.Time{}, false
	}
	return entry.Next, true
}
