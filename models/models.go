package models

type Short struct {
	Title     int    `json:"title" gorm:"integer;primary key"`
	VideoID   string `json:"videoid" gorm:"text;not null"`
	Revenue   int    `json:"revenue" gorm:"integer"`
	Expenses  int    `json:"expenses" gorm:"integer"`
	NetResult int    `json:"netresult" gorm:"integer"`
}
