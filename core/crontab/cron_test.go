package crontab

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

// 每 5 秒，6 段表达式。
const specEvery5s = "*/5 * * * * *"

// 语法合法但永远推算不出执行时间：2 月没有 31 日。
const specNeverFire = "0 0 0 31 2 *"

// 一个不依赖真实时钟的基准时间，便于做确定性断言。
var baseTime = time.Date(2026, 10, 12, 10, 0, 0, 0, time.Local)

func TestNewCron_SecondPrecision(t *testing.T) {
	c := NewCron()
	defer c.Stop()

	if _, err := c.AddFunc(specEvery5s, func() {}); err != nil {
		t.Fatalf("6 段表达式应被接受: %v", err)
	}

	// 5 段表达式在 WithSeconds() 下无效，必须报错而不是静默注册。
	if _, err := c.AddFunc("*/5 * * * *", func() {}); err == nil {
		t.Fatal("5 段表达式应被拒绝，实际通过了")
	}
}

func TestAddFunc_RegistersEntry(t *testing.T) {
	c := NewCron()
	defer c.Stop()

	id, err := c.AddFunc(specEvery5s, func() {})
	if err != nil {
		t.Fatalf("AddFunc: %v", err)
	}
	if id == 0 {
		t.Fatal("合法任务的 EntryID 不应为 0")
	}

	entries := c.Entries()
	if len(entries) != 1 {
		t.Fatalf("调度器里有 %d 个任务，期望 1 个", len(entries))
	}
	if entries[0].ID != id {
		t.Errorf("entry id = %v, 期望 %v", entries[0].ID, id)
	}
}

func TestAddFunc_RejectsBadSpec(t *testing.T) {
	c := NewCron()
	defer c.Stop()

	for _, spec := range []string{"", "not a cron", "*/5 * * * *", "61 * * * * *"} {
		id, err := c.AddFunc(spec, func() {})
		if err == nil {
			t.Errorf("表达式 %q 应被拒绝，实际通过了", spec)
		}
		// 失败时不应返回一个看起来可用的 id，否则调用方会拿它去查任务。
		if id != 0 {
			t.Errorf("表达式 %q 失败却返回了 id=%v", spec, id)
		}
	}

	// 全部失败后不应留下任何任务。
	if n := len(c.Entries()); n != 0 {
		t.Fatalf("非法表达式残留了 %d 个任务", n)
	}
}

func TestNextRun_ReturnsFutureTime(t *testing.T) {
	c := NewCron()
	defer c.Stop()

	// 未注册的 id：即便调度器已启动也应报告 ok=false。
	if _, ok := NextRun(c, 9999); ok {
		t.Fatal("未注册的 entry id 不应返回下次执行时间")
	}

	id, err := c.AddFunc(specEvery5s, func() {})
	if err != nil {
		t.Fatalf("AddFunc: %v", err)
	}

	// 未启动时 entry.Next 是零值，应报告 ok=false 而不是返回零值时间。
	if next, ok := NextRun(c, id); ok {
		t.Fatalf("启动前不应有下次执行时间，实际拿到 %s", next)
	}

	c.Start()

	deadline := time.Now().Add(2 * time.Second)
	for {
		next, ok := NextRun(c, id)
		if ok {
			if !next.After(time.Now()) {
				t.Fatalf("下次执行时间 %s 不在未来", next)
			}
			// 秒级表达式：触发点必须落在整秒上。
			if next.Nanosecond() != 0 {
				t.Errorf("下次执行时间 %s 未对齐整秒", next)
			}
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("启动后仍取不到下次执行时间")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// EntryID 0 是零值，永远不对应任何任务。
// 注意：即便去掉 NextRun 里的 id==0 显式判断，行为也仍是 ok=false
// （Entry(0) 返回零值 Entry），所以本用例锁定的是对外契约而非某段实现。
func TestNextRun_ZeroEntryID(t *testing.T) {
	c := NewCron()
	defer c.Stop()

	// 0 是 EntryID 的零值，永远不对应任何任务，无论调度器是否启动。
	if next, ok := NextRun(c, 0); ok {
		t.Fatalf("id=0 不应返回下次执行时间，实际拿到 %s", next)
	}

	// 即便调度器里已经存在合法任务，0 也依然无效。
	if _, err := c.AddFunc(specEvery5s, func() {}); err != nil {
		t.Fatalf("AddFunc: %v", err)
	}
	c.Start()

	if next, ok := NextRun(c, 0); ok {
		t.Fatalf("启动后 id=0 仍不应返回下次执行时间，实际拿到 %s", next)
	}
	// 合法 id（从 1 开始）此时应能取到时间，证明上面的 false 不是因为调度器没跑。
	if _, ok := NextRun(c, 1); !ok {
		t.Fatal("合法 id=1 应能取到下次执行时间")
	}
}

// 注册失败时返回的 id 必须是 0，调用方不应拿它去调用 NextRun。
func TestAddFunc_FailedIDIsZero(t *testing.T) {
	c := NewCron()
	defer c.Stop()

	id, err := c.AddFunc("bad spec", func() {})
	if err == nil {
		t.Fatal("非法表达式应返回错误")
	}
	if id != 0 {
		t.Fatalf("注册失败应返回 id=0, 实际 %v", id)
	}
	if _, ok := NextRun(c, id); ok {
		t.Fatal("失败注册返回的 id 不应能取到下次执行时间")
	}
}

func TestNextRun_TracksEntryAfterRemoval(t *testing.T) {
	c := NewCron()
	defer c.Stop()

	id, err := c.AddFunc(specEvery5s, func() {})
	if err != nil {
		t.Fatalf("AddFunc: %v", err)
	}
	c.Start()
	c.Remove(id)

	if _, ok := NextRun(c, id); ok {
		t.Fatal("已移除的任务不应返回下次执行时间")
	}
}

func TestParse_ReusesSchedule(t *testing.T) {
	p := NewCronParser()

	schedule, err := p.Parse(specEvery5s)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if _, err := p.Parse("bad spec"); err == nil {
		t.Fatal("非法表达式应返回错误")
	}

	// 自己控制起点，验证 schedule 可被反复使用。
	if got := schedule.Next(baseTime).Sub(baseTime); got != 5*time.Second {
		t.Fatalf("首次触发间隔 = %s, 期望 5s", got)
	}
}

func TestNextTimeListN_Count(t *testing.T) {
	p := NewCronParser()

	for _, n := range []int{1, 2, 5, 10} {
		times, err := p.NextTimeListN(specEvery5s, n)
		if err != nil {
			t.Fatalf("NextTimeListN(%d): %v", n, err)
		}
		if len(times) != n {
			t.Fatalf("NextTimeListN(%d) 返回 %d 个，期望 %d 个", n, len(times), n)
		}

		// 序列必须严格递增，且每步恰好 5 秒。
		for i := 1; i < len(times); i++ {
			if got := times[i].Sub(times[i-1]); got != 5*time.Second {
				t.Fatalf("第 %d 步间隔 = %s, 期望 5s", i, got)
			}
		}
		// 首个时间必须落在未来。
		if !times[0].After(time.Now().Add(-time.Second)) {
			t.Fatalf("首个执行时间 %s 不在未来", times[0])
		}
		// 不允许出现零值时间。
		for i, ts := range times {
			if ts.IsZero() {
				t.Fatalf("第 %d 个执行时间是零值", i)
			}
		}
	}
}

// 断言具体时间值，而不只是间隔：间隔正确但整体偏移错误的实现必须被拦住。
func TestNextTimeListN_ExactTimes(t *testing.T) {
	p := NewCronParser()

	// 使用可复用 schedule + 固定基准，得到确定性结果。
	// 字段顺序是「秒 分 时 日 月 周」，所以这是每小时的第 30 分钟整点。
	schedule, err := p.Parse("0 30 * * * *")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	want := []time.Time{
		time.Date(2026, 10, 12, 10, 30, 0, 0, time.Local),
		time.Date(2026, 10, 12, 11, 30, 0, 0, time.Local),
		time.Date(2026, 10, 12, 12, 30, 0, 0, time.Local),
	}
	at := baseTime
	for i, w := range want {
		got := schedule.Next(at)
		if !got.Equal(w) {
			t.Fatalf("第 %d 个执行时间 = %s, 期望 %s", i, got, w)
		}
		at = got
	}

	// 换一个只匹配整秒的表达式："第 30 秒"，验证秒字段确实生效。
	everyMinuteAt30s, err := p.Parse("30 * * * * *")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got := everyMinuteAt30s.Next(baseTime); !got.Equal(time.Date(2026, 10, 12, 10, 0, 30, 0, time.Local)) {
		t.Fatalf("第 30 秒表达式首次触发 = %s, 期望 2026-10-12 10:00:30", got)
	}

	// NextTimeListN 走真实时钟，只断言"分钟位为 30"这一确定性特征。
	times, err := p.NextTimeListN("0 30 * * * *", 3)
	if err != nil {
		t.Fatalf("NextTimeListN: %v", err)
	}
	for i, ts := range times {
		if ts.Minute() != 30 || ts.Second() != 0 {
			t.Fatalf("第 %d 个执行时间 %s 不是每小时的 30 分 0 秒", i, ts)
		}
	}
}

func TestNextTimeListN_NonPositiveCount(t *testing.T) {
	p := NewCronParser()

	for _, n := range []int{0, -1, -100} {
		times, err := p.NextTimeListN(specEvery5s, n)
		if err != nil {
			t.Fatalf("NextTimeListN(%d) 不应报错: %v", n, err)
		}
		if len(times) != 0 {
			t.Fatalf("NextTimeListN(%d) 返回 %d 个，期望 0 个", n, len(times))
		}
	}
}

func TestNextTimeListN_RejectsBadSpec(t *testing.T) {
	p := NewCronParser()

	_, err := p.NextTimeListN("*/5 * * * *", 3)
	if err == nil {
		t.Fatal("5 段表达式应返回错误")
	}
	// 语法错误不应被当成"无解"错误。
	if errors.Is(err, ErrNoNextTime) {
		t.Fatalf("语法错误不应匹配 ErrNoNextTime: %v", err)
	}
}

// 回归测试：语法合法但永不触发的表达式必须报错，
// 而不是静默返回一串 0001-01-01 的零值时间。
func TestNextTimeListN_NeverFiringSpec(t *testing.T) {
	p := NewCronParser()

	// 先确认表达式语法本身是合法的，否则这个用例测不到目标分支。
	if _, err := p.Parse(specNeverFire); err != nil {
		t.Fatalf("该表达式应语法合法: %v", err)
	}

	for _, n := range []int{1, 3, 5} {
		times, err := p.NextTimeListN(specNeverFire, n)
		if err == nil {
			t.Fatalf("永不触发的表达式应报错，实际返回 %v", times)
		}
		if !errors.Is(err, ErrNoNextTime) {
			t.Fatalf("错误应可用 errors.Is 匹配 ErrNoNextTime, 实际: %v", err)
		}
		if times != nil {
			t.Fatalf("报错时应返回 nil, 实际 %v", times)
		}
	}
}

// 永不触发的表达式同样不允许从 NextDateListN 静默渗出零值日期字符串。
func TestNextDateListN_NeverFiringSpec(t *testing.T) {
	p := NewCronParser()

	dates, err := p.NextDateListN(specNeverFire, 3)
	if err == nil {
		t.Fatalf("永不触发的表达式应报错，实际返回 %v", dates)
	}
	if !errors.Is(err, ErrNoNextTime) {
		t.Fatalf("错误应匹配 ErrNoNextTime, 实际: %v", err)
	}
}

// 旧方法必须保持既有行为：固定 5 个，与 NextTimeListN(spec, DefaultNextCount) 一致。
func TestNextTimeList_BackwardCompatible(t *testing.T) {
	p := NewCronParser()

	times, err := p.NextTimeList(specEvery5s)
	if err != nil {
		t.Fatalf("NextTimeList: %v", err)
	}
	if len(times) != DefaultNextCount {
		t.Fatalf("NextTimeList 返回 %d 个，期望 %d 个", len(times), DefaultNextCount)
	}

	dates, err := p.NextDateList(specEvery5s)
	if err != nil {
		t.Fatalf("NextDateList: %v", err)
	}
	if len(dates) != DefaultNextCount {
		t.Fatalf("NextDateList 返回 %d 个，期望 %d 个", len(dates), DefaultNextCount)
	}
	for _, d := range dates {
		if _, err := time.Parse(time.DateTime, d); err != nil {
			t.Fatalf("日期 %q 不是 time.DateTime 格式: %v", d, err)
		}
	}
}

// 旧方法也必须继承永不触发表达式的报错行为。
func TestNextTimeList_PropagatesNeverFiringError(t *testing.T) {
	p := NewCronParser()

	if _, err := p.NextTimeList(specNeverFire); !errors.Is(err, ErrNoNextTime) {
		t.Fatalf("NextTimeList 应返回 ErrNoNextTime, 实际: %v", err)
	}
	if _, err := p.NextDateList(specNeverFire); !errors.Is(err, ErrNoNextTime) {
		t.Fatalf("NextDateList 应返回 ErrNoNextTime, 实际: %v", err)
	}
}

func TestNextDateListN_Count(t *testing.T) {
	p := NewCronParser()

	dates, err := p.NextDateListN(specEvery5s, 3)
	if err != nil {
		t.Fatalf("NextDateListN: %v", err)
	}
	if len(dates) != 3 {
		t.Fatalf("NextDateListN 返回 %d 个，期望 3 个", len(dates))
	}
	for _, d := range dates {
		if _, err := time.Parse(time.DateTime, d); err != nil {
			t.Fatalf("日期 %q 不是 time.DateTime 格式: %v", d, err)
		}
	}
}

// n <= 0 在 NextDateListN 上同样应返回空列表而非报错（lo.Map 路径）。
func TestNextDateListN_NonPositiveCount(t *testing.T) {
	p := NewCronParser()

	for _, n := range []int{0, -1, -100} {
		dates, err := p.NextDateListN(specEvery5s, n)
		if err != nil {
			t.Fatalf("NextDateListN(%d) 不应报错: %v", n, err)
		}
		if len(dates) != 0 {
			t.Fatalf("NextDateListN(%d) 返回 %d 个，期望 0 个", n, len(dates))
		}
	}
}

// NextDateListN 必须把语法错误传播给调用方。
func TestNextDateListN_RejectsBadSpec(t *testing.T) {
	p := NewCronParser()

	if _, err := p.NextDateListN("*/5 * * * *", 3); err == nil {
		t.Fatal("5 段表达式应返回错误")
	}
}

// NewCron 的 Recover 链：任务 panic 不应拖垮调度器，后续任务照常执行。
func TestNewCron_RecoversFromPanic(t *testing.T) {
	c := NewCron()
	defer c.Stop()

	var survived atomic.Bool
	if _, err := c.AddFunc("*/1 * * * * *", func() { panic("boom") }); err != nil {
		t.Fatalf("AddFunc: %v", err)
	}
	// 另一个任务必须在同一次触发中存活下来。
	if _, err := c.AddFunc("*/1 * * * * *", func() { survived.Store(true) }); err != nil {
		t.Fatalf("AddFunc: %v", err)
	}

	c.Start()

	deadline := time.Now().Add(3 * time.Second)
	for !survived.Load() {
		if time.Now().After(deadline) {
			t.Fatal("panic 任务影响了同批次其它任务的执行")
		}
		time.Sleep(20 * time.Millisecond)
	}

	// 调度器仍在运行，没有因为 panic 而停止。
	if _, ok := NextRun(c, 1); !ok {
		t.Fatal("panic 后调度器应仍在运行")
	}
}

// NewCron 的 DelayIfStillRunning 链：慢任务不会被并发重入。
func TestNewCron_DelayIfStillRunning(t *testing.T) {
	c := NewCron()
	defer c.Stop()

	var running, maxConcurrent atomic.Int32
	if _, err := c.AddFunc("*/1 * * * * *", func() {
		cur := running.Add(1)
		for {
			old := maxConcurrent.Load()
			if cur <= old || maxConcurrent.CompareAndSwap(old, cur) {
				break
			}
		}
		time.Sleep(1500 * time.Millisecond)
		running.Add(-1)
	}); err != nil {
		t.Fatalf("AddFunc: %v", err)
	}

	c.Start()

	// 任务耗时 1.5s > 触发间隔 1s，若未串行化必然出现并发。
	time.Sleep(3200 * time.Millisecond)
	c.Stop()

	if got := maxConcurrent.Load(); got > 1 {
		t.Fatalf("同一任务出现 %d 次并发执行，DelayIfStillRunning 未生效", got)
	}
}
