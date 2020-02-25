package testutil

import (
	"context"
	"crypto/rand"
	"fmt"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/storage"
	"github.com/gofrs/uuid"
	"golang.org/x/crypto/bcrypt"
	"time"
)

const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func RegisterBrokerPlatformCredentials(repository storage.Repository, brokerID, platformID string) (string, string) {
	username := generateCredential()
	password := generateCredential()

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		panic(err)
	}

	credentialID, err := uuid.NewV4()
	if err != nil {
		panic(fmt.Sprintf("failed to generate instance GUID: %s", err))
	}

	credentials := &types.BrokerPlatformCredential{
		Base: types.Base{
			ID:        credentialID.String(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Ready:     true,
		},
		PlatformID:   platformID,
		BrokerID:     brokerID,
		Username:     username,
		PasswordHash: string(passwordHash),
	}

	if _, err := repository.Create(context.TODO(), credentials); err != nil {
		panic(err)
	}

	return username, password
}

func generateCredential() string {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		panic(fmt.Sprintf("could not generate credential: %v", err))
	}

	for idx, b := range bytes {
		bytes[idx] = letters[b%byte(len(letters))]
	}

	return string(bytes)
}
