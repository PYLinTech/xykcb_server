package schools

import (
	"bytes"
	"crypto/aes"
	"encoding/base64"
	"encoding/hex"
	"fmt"

	"xykcb_server/internal/cache"
	"xykcb_server/internal/httpclient"
)

func (s *HnitA) retryWithValidToken(account, password, path string, fetch func(path string) (map[string]interface{}, error)) (map[string]interface{}, error) {
	data, err := fetch(path)
	if err != nil {
		return nil, err
	}

	code := safeString(data["code"], "")
	if code != "" && code != "1" {
		cache.InvalidateToken(s.GetProviderKey(), account)
		token, err := s.getToken(account, password)
		if err != nil {
			return nil, err
		}
		replacedPath := httpclient.ReplaceTokenInPath(path, token)
		return fetch(replacedPath)
	}

	return data, nil
}

func (s *HnitA) encryptPassword(password string) string {
	key, _ := hex.DecodeString("717a6b6a316b6a6768643d383736262a")
	block, _ := aes.NewCipher(key)
	plain := []byte("\"" + password + "\"")
	size := block.BlockSize()
	padding := size - len(plain)%size
	plain = append(plain, bytes.Repeat([]byte{byte(padding)}, padding)...)
	encrypted := make([]byte, len(plain))
	for i := 0; i < len(plain); i += size {
		block.Encrypt(encrypted[i:i+size], plain[i:i+size])
	}
	return base64.StdEncoding.EncodeToString([]byte(base64.StdEncoding.EncodeToString(encrypted)))
}

func (s *HnitA) getToken(account, password string) (string, error) {
	return cache.GetToken(s.GetProviderKey(), account, password, func(account, password string) (string, error) {
		resp, err := schoolClient.Post("/login?userNo="+account+"&pwd="+s.encryptPassword(password), "")
		if err != nil {
			return "", err
		}

		code := safeString(resp["code"], "")
		if code != "1" {
			return "", fmt.Errorf("%s", safeString(resp["Msg"], "login failed"))
		}

		token := safeStringMap(resp["data"], "token")
		if token == "" {
			return "", fmt.Errorf("no token in response")
		}

		return token, nil
	})
}
