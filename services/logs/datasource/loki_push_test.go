package datasource

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestParseTimestampWindow(t *testing.T) {
	lower, upper := parseTimestampWindow(`entry too far behind, entry timestamp is: 2026-10-16T04:12:22Z, oldest acceptable timestamp is: 2026-10-16T04:15:26Z`)
	if lower.IsZero() || !upper.IsZero() {
		t.Fatalf("区间解析错误: lower=%s upper=%s", lower, upper)
	}
	if got := lower.Format(time.RFC3339); got != "2026-10-16T04:15:27Z" {
		t.Errorf("下界应留 1 秒余量: %s", got)
	}

	lower, upper = parseTimestampWindow(`entry too far ahead, newest acceptable timestamp is: 2026-10-16T04:15:26Z`)
	if !lower.IsZero() || upper.Format(time.RFC3339) != "2026-10-16T04:15:25Z" {
		t.Errorf("上界解析错误: lower=%s upper=%s", lower, upper)
	}

	if lower, upper := parseTimestampWindow("max length of a line is 1024"); !lower.IsZero() || !upper.IsZero() {
		t.Error("无关拒绝原因不应解析出时间区间")
	}
}

func TestClampStreams(t *testing.T) {
	lower := time.Date(2026, 10, 16, 4, 0, 0, 0, time.UTC)
	upper := time.Date(2026, 10, 16, 6, 0, 0, 0, time.UTC)
	streams := []Stream{{Labels: map[string]string{"job": "ops_log"}, Entries: []LogEntry{
		{Line: "old", Timestamp: lower.Add(-time.Hour)},
		{Line: "in", Timestamp: lower.Add(time.Minute)},
		{Line: "future", Timestamp: upper.Add(time.Hour)},
	}}}

	clamped := clampStreams(streams, lower, time.Time{})
	if got := clamped[0].Entries[0].Timestamp; !got.Equal(lower) {
		t.Errorf("过旧行未钳制到下界: %s", got)
	}
	if got := clamped[0].Entries[1].Timestamp; !got.Equal(lower.Add(time.Minute)) {
		t.Error("区间内的行不应被改动")
	}
	if clamped[0].Entries[0].Line != "old" {
		t.Error("钳制不应改动日志内容")
	}
	if clamped[0].Labels["job"] != "ops_log" {
		t.Error("钳制后标签应保持不变")
	}

	both := clampStreams(streams, time.Time{}, upper)
	if got := both[0].Entries[2].Timestamp; !got.Equal(upper) {
		t.Errorf("过新行未钳制到上界: %s", got)
	}

	// 原始入参不应被就地修改
	if !streams[0].Entries[0].Timestamp.Equal(lower.Add(-time.Hour)) {
		t.Error("clampStreams 修改了调用方的数据")
	}
}

func newPushServer(t *testing.T, lookback time.Duration, reject string, attempts *atomic.Int32) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if attempts != nil {
			attempts.Add(1)
		}
		if reject != "" {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, reject)
			return
		}
		var payload struct {
			Streams []struct {
				Values [][]string `json:"values"`
			} `json:"streams"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, err.Error())
			return
		}
		accepted := time.Now().Add(-lookback)
		for _, stream := range payload.Streams {
			for _, value := range stream.Values {
				ns, err := strconv.ParseInt(value[0], 10, 64)
				if err != nil {
					continue
				}
				if time.Unix(0, ns).Before(accepted) {
					w.WriteHeader(http.StatusBadRequest)
					fmt.Fprintf(w, "entry too far behind, oldest acceptable timestamp is: %s",
						accepted.UTC().Format(time.RFC3339))
					return
				}
			}
		}
		w.WriteHeader(http.StatusNoContent)
	}))
}

func TestPushClampsWhenTimestampRejected(t *testing.T) {
	var attempts atomic.Int32
	server := newPushServer(t, time.Hour, "", &attempts)
	defer server.Close()

	client := NewClient()
	client.base = server.URL

	if err := client.Push(context.Background(), []Stream{{
		Labels:  map[string]string{"job": "ops_log_clamp_test"},
		Entries: []LogEntry{{Line: "历史日志", Timestamp: time.Now().Add(-3 * time.Hour)}},
	}}); err != nil {
		t.Fatalf("钳制重推后应写入成功: %v", err)
	}
	if attempts.Load() != 2 {
		t.Errorf("应只重试一次: attempts=%d", attempts.Load())
	}
}

func TestPushKeepsErrorForUnrelatedRejection(t *testing.T) {
	var attempts atomic.Int32
	server := newPushServer(t, 0, "Log entry is too long", &attempts)
	defer server.Close()

	client := NewClient()
	client.base = server.URL

	err := client.Push(context.Background(), []Stream{{
		Labels:  map[string]string{"job": "ops_log_reject_test"},
		Entries: []LogEntry{{Line: "x", Timestamp: time.Now()}},
	}})
	if err == nil {
		t.Fatal("无关拒绝应返回错误，让采集端回退位点")
	}
	if !strings.Contains(err.Error(), "Log entry is too long") {
		t.Errorf("错误信息应保留 Loki 原文: %v", err)
	}
	if attempts.Load() != 1 {
		t.Errorf("无关拒绝不应重复投递: attempts=%d", attempts.Load())
	}
}
