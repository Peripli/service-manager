// Package types defines the entity types used in the Service Manager
package types

// Broker Just to showcase how to use
type Broker struct {
	ID        string `db:"id"`
	Name      string `db:"name"`
	URL       string `db:"broker_url"`
	CreatedAt string `db:"created_at"`
	UpdatedAt string `db:"updated_at"`
	User      string `db:"user"`
	Password  string `db:"password"`
}
