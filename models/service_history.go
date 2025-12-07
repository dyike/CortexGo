package models

// HistoryParams 描述查询历史报告的参数
type HistoryParams struct {
	Symbol    string `json:"symbol"`
	TradeDate string `json:"trade_date"`
	Cursor    string `json:"cursor"` // 书签分页用
	Limit     int    `json:"limit"`  // 每页数量，默认 50，最大 200
}

// HistoryFile 表示单个 Markdown 报告文件
type HistoryFile struct {
	Name    string `json:"name"`
	Path    string `json:"path"` // 相对 results 目录的路径，便于做书签
	Content string `json:"content"`
}
