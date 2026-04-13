package handler

// rawJSON 用于在日志中输出原始 JSON，不带转义
type rawJSON []byte

func (r rawJSON) String() string { return string(r) }
