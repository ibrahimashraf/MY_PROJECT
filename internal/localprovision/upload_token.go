package localprovision

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Provision-bound upload tokens let field devices authenticate TUS media
// uploads without an interactive OIDC login. The token is minted alongside
// the device authority at provision time, bound to that exact
// (device, authority, epoch) triple, and expires with the authority — so a
// rotated authority automatically invalidates previously issued tokens.
// Validation is stateless (HMAC with the server sync secret); no database
// lookup is required.
const (
	uploadTokenVersion = "v2"
	uploadTokenDomain  = "integin-upload-token-v2"
	uploadTokenSkew    = 60 * time.Second
)

func MintUploadToken(secret, tenantID, deviceID, authorityID, sessionID string, epoch uint64, expiresAt time.Time) (string, error) {
	if strings.TrimSpace(secret) == "" {
		return "", errors.New("upload token requires a signing secret")
	}
	if strings.TrimSpace(deviceID) == "" || strings.TrimSpace(authorityID) == "" || strings.TrimSpace(tenantID) == "" {
		return "", errors.New("upload token requires tenant, device and authority ids")
	}
	payload := strings.Join([]string{
		tenantID,
		deviceID,
		authorityID,
		sessionID,
		strconv.FormatUint(epoch, 10),
		strconv.FormatInt(expiresAt.UTC().Unix(), 10),
	}, "|")
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(uploadTokenDomain + "|" + payload))
	token := uploadTokenVersion + "." +
		base64.RawURLEncoding.EncodeToString([]byte(payload)) + "." +
		base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return token, nil
}

type UploadTokenClaims struct {
	TenantID    string
	DeviceID    string
	AuthorityID string
	SessionID   string
	Epoch       uint64
	ExpiresAt   time.Time
}

func ValidateUploadToken(secret, token string, now time.Time) (UploadTokenClaims, error) {
	var claims UploadTokenClaims
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return claims, errors.New("upload token has an invalid format")
	}
	ver := parts[0]

	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return claims, errors.New("upload token payload is not valid base64url")
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return claims, errors.New("upload token signature is not valid base64url")
	}

	mac := hmac.New(sha256.New, []byte(secret))
	if ver == "v2" {
		_, _ = mac.Write([]byte("integin-upload-token-v2|" + string(payloadBytes)))
	} else if ver == "v1" {
		_, _ = mac.Write([]byte("integin-upload-token-v1|" + string(payloadBytes)))
	} else {
		return claims, errors.New("unsupported token version")
	}

	if subtle.ConstantTimeCompare(mac.Sum(nil), signature) != 1 {
		return claims, errors.New("upload token signature mismatch")
	}

	fields := strings.Split(string(payloadBytes), "|")
	if ver == "v2" {
		if len(fields) != 6 || fields[0] == "" || fields[1] == "" || fields[2] == "" {
			return claims, errors.New("upload token payload is malformed")
		}
		epoch, err := strconv.ParseUint(fields[4], 10, 64)
		if err != nil {
			return claims, errors.New("upload token epoch is malformed")
		}
		expiresUnix, err := strconv.ParseInt(fields[5], 10, 64)
		if err != nil {
			return claims, errors.New("upload token expiry is malformed")
		}
		expiresAt := time.Unix(expiresUnix, 0).UTC()
		if now.After(expiresAt.Add(uploadTokenSkew)) {
			return claims, fmt.Errorf("upload token expired at %s", expiresAt.Format(time.RFC3339))
		}

		return UploadTokenClaims{
			TenantID:    fields[0],
			DeviceID:    fields[1],
			AuthorityID: fields[2],
			SessionID:   fields[3],
			Epoch:       epoch,
			ExpiresAt:   expiresAt,
		}, nil
	} else {
		if len(fields) != 4 || fields[0] == "" || fields[1] == "" {
			return claims, errors.New("upload token payload is malformed")
		}
		epoch, err := strconv.ParseUint(fields[2], 10, 64)
		if err != nil {
			return claims, errors.New("upload token epoch is malformed")
		}
		expiresUnix, err := strconv.ParseInt(fields[3], 10, 64)
		if err != nil {
			return claims, errors.New("upload token expiry is malformed")
		}
		expiresAt := time.Unix(expiresUnix, 0).UTC()
		if now.After(expiresAt.Add(uploadTokenSkew)) {
			return claims, fmt.Errorf("upload token expired at %s", expiresAt.Format(time.RFC3339))
		}

		return UploadTokenClaims{
			DeviceID:    fields[0],
			AuthorityID: fields[1],
			Epoch:       epoch,
			ExpiresAt:   expiresAt,
		}, nil
	}
}
