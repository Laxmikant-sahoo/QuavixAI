package chat


type Chat struct {
	ID     string `db:"id" json:"id"`
	UserID string `db:"user_id" json:"user_id"`
	Title  string `db:"title" json:"title"`
}
