package models

// HistoryParams 描述查询历史列表的参数（书签分页）
type HistoryParams struct {
	Cursor string `json:"cursor"` // 书签分页用
	Limit  int    `json:"limit"`  // 每页数量，默认 50，最大 200
}

// HistoryListItem 表示列表返回的目录/文件信息（无内容）
type HistoryListItem struct {
	Name string `json:"name"`
	Path string `json:"path"` // 相对 results 目录的路径，便于书签
}

// HistoryFile 表示单个 Markdown 报告文件（含内容）
type HistoryFile struct {
	Name    string `json:"name"`
	Path    string `json:"path"` // 相对 results 目录的路径，便于书签
	Content string `json:"content"`
}

// HistoryInfoParams 描述按路径读取历史详情的参数
type HistoryInfoParams struct {
	Path string `json:"path"` // 相对 results 目录的路径，可以是目录或单个 .md 文件
}
