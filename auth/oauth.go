package auth

import (
	"fmt"
	"net/http"

	"golang.org/x/net/context"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/jwt"
)

func NewServiceAccountFile(serviceAccountFile string) ([]byte, error) {
	content, exists, err := ReadFile(serviceAccountFile)
	if !exists {
		return nil, fmt.Errorf("service account filename %q not found", serviceAccountFile)
	}

	if err != nil {
		return nil, err
	}
	return content, nil
}

func NewServiceAccount(content []byte) (*jwt.Config, error) {
	conf, err := google.JWTConfigFromJSON(content, "https://www.googleapis.com/auth/drive")
	if err != nil {
		return nil, err
	}
	return conf, nil
}

func NewServiceAccountClient(ctx context.Context, j *jwt.Config) *http.Client {
	return j.Client(ctx)
}
