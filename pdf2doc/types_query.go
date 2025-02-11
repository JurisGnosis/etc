package pdf2doc

type queryResponse struct {
	Code int               `json:"code"`
	Data queryResponseData `json:"data"`
}

type queryResponseData struct {
	Status      int     `json:"status"`
	Duration    float64 `json:"duration"`
	DownloadURL string  `json:"download_url"`
	TaskID      string  `json:"task_id"`
	Progress    int     `json:"progress"`
	StartTime   int64   `json:"start_time"`
	PageCount   int     `json:"page_count"`
	ErrMsgs     *string `json:"errMsgs"` // 使用指针表示可能为 null
}
