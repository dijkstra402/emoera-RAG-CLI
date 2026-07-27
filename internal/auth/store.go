package auth

import (
	"errors"
	"strings"

	"github.com/dijkstra402/emoera-RAG-CLI/internal/apperr"
	"github.com/zalando/go-keyring"
)

const serviceName = "emoera-cli"

type Store interface {
	Get(profile string) (string, error)
	Set(profile, token string) error
	Delete(profile string) error
}

type KeyringStore struct{}

func (KeyringStore) Get(profile string) (string, error) {
	token, err := keyring.Get(serviceName, profile)
	if errors.Is(err, keyring.ErrNotFound) {
		return "", nil
	}
	if err != nil {
		return "", apperr.Wrap(apperr.ExitConfiguration, "读取系统 Keychain 失败", err)
	}
	return token, nil
}

func (KeyringStore) Set(profile, token string) error {
	if strings.TrimSpace(token) == "" {
		return apperr.New(apperr.ExitArguments, "Token 不能为空")
	}
	if err := keyring.Set(serviceName, profile, strings.TrimSpace(token)); err != nil {
		return apperr.Wrap(apperr.ExitConfiguration, "写入系统 Keychain 失败", err)
	}
	return nil
}

func (KeyringStore) Delete(profile string) error {
	err := keyring.Delete(serviceName, profile)
	if errors.Is(err, keyring.ErrNotFound) {
		return nil
	}
	if err != nil {
		return apperr.Wrap(apperr.ExitConfiguration, "删除系统 Keychain 凭据失败", err)
	}
	return nil
}
