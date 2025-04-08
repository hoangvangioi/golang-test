package models

// Dialog struct represents the 'dialog' table
type Dialog struct {
	ID      int64
	Lang    string
	Content string
}

// Word struct represents the 'word' table
type Word struct {
	ID        int64
	Lang      string
	Content   string
	Translate string
}

// WordDialog struct represents the 'word_dialog' table (no explicit fields needed as it's a join table)
type WordDialog struct {
	DialogID int64
	WordID   int64
}
