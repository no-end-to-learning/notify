package service

type Channel string

const (
	ChannelLark     Channel = "lark"
	ChannelTelegram Channel = "telegram"
)

type Color string

const (
	ColorBlue   Color = "Blue"
	ColorGreen  Color = "Green"
	ColorOrange Color = "Orange"
	ColorGrey   Color = "Grey"
	ColorRed    Color = "Red"
	ColorPurple Color = "Purple"
)

type MessageParams struct {
	Title   string `json:"title,omitempty"`
	Color   Color  `json:"color,omitempty"`
	Content string `json:"content,omitempty"`
	URL     string `json:"url,omitempty"`
	Note    string `json:"note,omitempty"`
}

type SendResult struct {
	TaskID  string `json:"taskId,omitempty"`
	Success bool   `json:"success"`
}

type ChatItem struct {
	ChatID      string `json:"chatId"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type GrafanaAlert struct {
	State       string      `json:"state"`
	RuleName    string      `json:"ruleName"`
	RuleURL     string      `json:"ruleUrl,omitempty"`
	Message     string      `json:"message,omitempty"`
	ImageURL    string      `json:"imageUrl,omitempty"`
	EvalMatches []EvalMatch `json:"evalMatches,omitempty"`
}

type EvalMatch struct {
	Metric string  `json:"metric"`
	Value  float64 `json:"value"`
}

var ColorEmoji = map[Color]string{
	ColorBlue:   "ℹ️",
	ColorGreen:  "✅",
	ColorOrange: "⚠️",
	ColorGrey:   "⏸️",
	ColorRed:    "❌",
	ColorPurple: "🔮",
}
