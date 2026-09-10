package crontab

import (
	"log/slog"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/samber/lo"
)

type CronParser struct {
	Parser cron.Parser
}

func NewCronParser() *CronParser {
	return &CronParser{
		Parser: cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow),
	}
}

// 解析下一个执行时间列表
func (p *CronParser) NextTimeList(spec string) (nextTimes []time.Time, err error) {
	schedule, err := p.Parser.Parse(spec)
	if err != nil {
		slog.Error("cron parser parse next error",
			slog.String("spec", spec),
			slog.Any("error", err),
		)
		return nil, err
	}

	now := time.Now()
	for i := 0; i < 5; i++ {
		next := schedule.Next(now)
		nextTimes = append(nextTimes, next)
		now = next
	}
	return nextTimes, nil
}

// 解析下一个执行日期列表
func (p *CronParser) NextDateList(spec string) ([]string, error) {
	nextTimes, err := p.NextTimeList(spec)
	if err != nil {
		return nil, err
	}
	return lo.Map(nextTimes, func(nextTime time.Time, _ int) string {
		return nextTime.Format(time.DateTime)
	}), nil
}
