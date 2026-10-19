package api

// ImportTemplateResponse 导入模板下载响应，模板内容以 base64 返回，前端还原为文件下载
type ImportTemplateResponse struct {
	FileName string `json:"fileName"`
	Content  string `json:"content"`
}

// ImportRowError 单行导入失败明细
type ImportRowError struct {
	Row    int    `json:"row"`
	Name   string `json:"name"`
	Reason string `json:"reason"`
}

// ImportResult 批量导入结果
type ImportResult struct {
	Total   int              `json:"total"`
	Success int              `json:"success"`
	Failed  int              `json:"failed"`
	Errors  []ImportRowError `json:"errors"`
}
