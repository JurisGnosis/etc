package ryconn

// Response 代表整个 JSON 响应的结构
type RuoyiResponse struct {
	Msg  string        `json:"msg"`
	Code int           `json:"code"`
	Data RuoyiUserData `json:"data"`
}

// UserData 代表 data 部分的结构
type RuoyiUserData struct {
	ID          int         `json:"id"`
	MediateID   int         `json:"mediateId"`
	LawyerID    int         `json:"lawyerId"`
	Type        string      `json:"type"`
	Types       interface{} `json:"types"`    // 因为 types 是 null，这里用 interface{} 表示
	Username    *string     `json:"username"` // 可能为 null，使用指针类型
	Password    *string     `json:"password"` // 可能为 null，使用指针类型
	Nickname    string      `json:"nickname"`
	Avatar      string      `json:"avatar"`
	Mobile      string      `json:"mobile"`
	LoginLimit  *int        `json:"loginLimit"`  // 可能为 null，使用指针类型
	LastLoginAt *string     `json:"lastLoginAt"` // 可能为 null，使用指针类型
	CreatedAt   string      `json:"createdAt"`
	UpdatedAt   string      `json:"updatedAt"`
	Banners     interface{} `json:"banners"` // 因为 banners 是 null，这里用 interface{} 表示
	ChannelID   int         `json:"channelId"`
}
