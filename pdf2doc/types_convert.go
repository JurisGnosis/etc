package pdf2doc

type convertResponse struct {
	Code int `json:"code"`
	Data struct {
		TaskId string `json:"task_id"`
	} `json:"data"`
}
