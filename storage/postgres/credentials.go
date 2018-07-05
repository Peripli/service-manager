package postgres

import (
	"github.com/jmoiron/sqlx"
	"fmt"
	"github.com/Peripli/service-manager/types"
)

type credentialStorage struct {
	db *sqlx.DB
}

func (cs *credentialStorage) Get(username string) (*types.Credentials, error) {
	credentials := &Credentials{}
	query := fmt.Sprintf(`SELECT c.username "username",
 								 c.password "password"
							FROM %s AS c JOIN %s as p ON c.id=p.credentials_id
							WHERE c.username=$1`, credentialsTable, platformTable)

	err := cs.db.Get(credentials, query, username)

	if err != nil {
		return nil, checkSQLNoRows(err)
	}

	return &types.Credentials{
		Basic: &types.Basic{
			Username: credentials.Username,
			Password: credentials.Password,
		},
	}, nil
}