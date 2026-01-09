package auth

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"image/png"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

var (
	ErrInvalidTOTPCode = errors.New("invalid TOTP code")
)

type TOTPService interface {
	GenerateSecret(ctx context.Context, user *model.User) (string, string, error)
	ValidateCode(ctx context.Context, secret string, code string) bool
	GenerateQRCode(ctx context.Context, user *model.User, secret string) (string, error)
}

type totpService struct{}

func NewTOTPService() TOTPService {
	return &totpService{}
}

// GenerateSecret generates a new TOTP secret for a user
func (s *totpService) GenerateSecret(ctx context.Context, user *model.User) (string, string, error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "Navidrome",
		AccountName: user.UserName,
	})
	if err != nil {
		log.Error(ctx, "Failed to generate TOTP secret", "user", user.UserName, err)
		return "", "", err
	}

	return key.Secret(), key.URL(), nil
}

// ValidateCode validates a TOTP code against a secret
func (s *totpService) ValidateCode(ctx context.Context, secret string, code string) bool {
	valid := totp.Validate(code, secret)
	if !valid {
		log.Warn(ctx, "Invalid TOTP code provided")
	}
	return valid
}

// GenerateQRCode generates a base64-encoded PNG QR code for the TOTP setup
func (s *totpService) GenerateQRCode(ctx context.Context, user *model.User, secret string) (string, error) {
	key, err := otp.NewKeyFromURL(secret)
	if err != nil {
		log.Error(ctx, "Failed to create key from URL", err)
		return "", err
	}

	// Generate QR code image
	img, err := key.Image(200, 200)
	if err != nil {
		log.Error(ctx, "Failed to generate QR code image", err)
		return "", err
	}

	// Encode to PNG
	var buf bytes.Buffer
	err = png.Encode(&buf, img)
	if err != nil {
		log.Error(ctx, "Failed to encode QR code as PNG", err)
		return "", err
	}

	// Convert to base64
	encoded := base64.StdEncoding.EncodeToString(buf.Bytes())
	return "data:image/png;base64," + encoded, nil
}
