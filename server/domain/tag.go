package domain

type (
	TagAddRequest struct {
		Name    string `json:"name" binding:"required"`
		Color   string `json:"color"`
		BgColor string `json:"bgColor"`
	}
	TagOrderRequest struct {
		TagIDs []uint `json:"tagIds" binding:"required"`
	}
)
