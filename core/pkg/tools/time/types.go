package time

import "time"

const (
	ToolGetCurrentTime = "get_current_time"
	ToolConvertTime    = "convert_time"
)

type TimeResult struct {
	Timezone  string `json:"timezone"`
	Datetime  string `json:"datetime"`
	DayOfWeek string `json:"day_of_week"`
	IsDST     bool   `json:"is_dst"`
}

type TimeConversionResult struct {
	Source         TimeResult `json:"source"`
	Target         TimeResult `json:"target"`
	TimeDifference string     `json:"time_difference"`
}

type TimeConversionInput struct {
	SourceTimezone string `json:"source_timezone"`
	Time           string `json:"time"`
	TargetTimezone string `json:"target_timezone"`
}

func formatTimeResult(t time.Time, timezone string) TimeResult {
	return TimeResult{
		Timezone:  timezone,
		Datetime:  t.Format(time.RFC3339),
		DayOfWeek: t.Format("Monday"),
		IsDST:     t.IsDST(),
	}
}
