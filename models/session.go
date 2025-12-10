package models

type SessionRecord struct {
	ID        string
	Symbol    string
	TradeDate string
	Prompt    string
	Status    string
}

type MessageRecord struct {
	ID           string
	SessionID    string
	Role         string
	Agent        string
	Content      string
	Status       string
	FinishReason string
	Seq          int
}

type SessionWithMeta struct {
	SessionRecord
	RowID     int64
	CreatedAt string
	UpdatedAt string
}

type MessageWithMeta struct {
	MessageRecord
	CreatedAt string
	UpdatedAt string
}
