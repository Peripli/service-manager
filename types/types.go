package types

type Broker struct {
	ID        string `db:"id"`
	Name      string `db:"name"`
	URL       string `db:"broker_url"`
	CreatedAt string `db:"created_at"`
	UpdatedAt string `db:"updated_at"`
	User      string `db:"user"`
	Password  string `db:"password"`
}
