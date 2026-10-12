package crontab

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/samber/lo"
)

// ErrNoNextTime 表示表达式语法合法，但永远推算不出执行时间。
// 调用方可用 errors.Is 区分它和语法错误。
var ErrNoNextTime = errors.New("cron: 表达式推算不出执行时间")

type CronParser struct {
	Parser cron.Parser
}

func NewCronParser() *CronParser {
	return &CronParser{
		Parser: cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow),
	}
}

// DefaultNextCount 是不指定次数时返回的执行时间个数。
const DefaultNextCount = 5

// Parse 解析 cron 表达式，出错时记一条日志并返回 error。
//
// 需要复用同一个 schedule 做多次推算（例如自己控制起点）时用这个，
// 可以省掉重复解析；只是取几次时间的话直接用下面的 NextTimeList 即可。
func (p *CronParser) Parse(spec string) (cron.Schedule, error) {
	schedule, err := p.Parser.Parse(spec)
	if err != nil {
		slog.Error("cron parser parse next error",
			slog.String("spec", spec),
			slog.Any("error", err),
		)
		return nil, err
	}
	return schedule, nil
}

// NextTimeListN 解析接下来 n 个执行时间，从当前时刻开始连续推算。
//
// n <= 0 时返回空列表且不报错；表达式语法非法、或语法合法但永远推算不出
// 执行时间（例如 "0 0 0 31 2 *"，2 月没有 31 日）时返回错误。
func (p *CronParser) NextTimeListN(spec string, n int) ([]time.Time, error) {
	if n <= 0 {
		return []time.Time{}, nil
	}

	schedule, err := p.Parse(spec)
	if err != nil {
		return nil, err
	}

	nextTimes := make([]time.Time, 0, n)
	now := time.Now()
	for i := 0; i < n; i++ {
		next := schedule.Next(now)
		// 语法合法但无解时 cron 返回零值时间。若放任它往下走，会连续
		// 产出 n 个 0001-01-01，调用方拿到的是一个看似正常的列表，
		// 这里必须显式报错，避免把"永不触发"伪装成有效结果。
		if next.IsZero() {
			return nil, fmt.Errorf("%w: spec %q 推算不出执行时间", ErrNoNextTime, spec)
		}
		// 以上一次的结果为下一次的起点，保证序列严格递增。
		nextTimes = append(nextTimes, next)
		now = next
	}
	return nextTimes, nil
}

// NextDateListN 解析接下来 n 个执行日期，格式为 time.DateTime。
func (p *CronParser) NextDateListN(spec string, n int) ([]string, error) {
	nextTimes, err := p.NextTimeListN(spec, n)
	if err != nil {
		return nil, err
	}
	return lo.Map(nextTimes, func(nextTime time.Time, _ int) string {
		return nextTime.Format(time.DateTime)
	}), nil
}

// 解析下一个执行时间列表，固定返回 DefaultNextCount 个。
func (p *CronParser) NextTimeList(spec string) ([]time.Time, error) {
	return p.NextTimeListN(spec, DefaultNextCount)
}

// 解析下一个执行日期列表，固定返回 DefaultNextCount 个。
func (p *CronParser) NextDateList(spec string) ([]string, error) {
	return p.NextDateListN(spec, DefaultNextCount)
}
