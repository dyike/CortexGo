package models

import "time"

type SessionRecord struct {
	Id        int64
	Symbol    string
	TradeDate string
	Prompt    string
	Status    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type MessageRecord struct {
	Id           int64
	SessionId    int64
	Role         string
	Agent        string
	Content      string
	Status       string
	FinishReason string
	Seq          int
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
