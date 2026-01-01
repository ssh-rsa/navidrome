package auth

import (
	"context"
	"testing"
	"time"

	"github.com/navidrome/navidrome/model"
	"github.com/pquerna/otp/totp"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Helper function to get current time for TOTP generation
func Now() time.Time {
	return time.Now()
}

func TestTOTP(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "TOTP Suite")
}

var _ = Describe("TOTPService", func() {
	var service TOTPService
	var ctx context.Context
	var testUser *model.User

	BeforeEach(func() {
		service = NewTOTPService()
		ctx = context.Background()
		testUser = &model.User{
			ID:       "user-1",
			UserName: "testuser",
			Name:     "Test User",
			Email:    "test@example.com",
		}
	})

	Describe("GenerateSecret", func() {
		It("should generate a valid TOTP secret", func() {
			secret, url, err := service.GenerateSecret(ctx, testUser)

			Expect(err).ToNot(HaveOccurred())
			Expect(secret).ToNot(BeEmpty())
			Expect(url).To(ContainSubstring("otpauth://totp/"))
			Expect(url).To(ContainSubstring("Navidrome"))
			Expect(url).To(ContainSubstring("testuser"))
		})
	})

	Describe("ValidateCode", func() {
		It("should validate a correct TOTP code", func() {
			secret, _, err := service.GenerateSecret(ctx, testUser)
			Expect(err).ToNot(HaveOccurred())

			// Generate a valid code using the secret
			code, err := totp.GenerateCode(secret, Now())
			Expect(err).ToNot(HaveOccurred())

			// Validate the code
			valid := service.ValidateCode(ctx, secret, code)
			Expect(valid).To(BeTrue())
		})

		It("should reject an invalid TOTP code", func() {
			secret, _, err := service.GenerateSecret(ctx, testUser)
			Expect(err).ToNot(HaveOccurred())

			// Try an invalid code
			valid := service.ValidateCode(ctx, secret, "000000")
			Expect(valid).To(BeFalse())
		})
	})

	Describe("GenerateQRCode", func() {
		It("should generate a base64-encoded QR code", func() {
			_, url, err := service.GenerateSecret(ctx, testUser)
			Expect(err).ToNot(HaveOccurred())

			qrCode, err := service.GenerateQRCode(ctx, testUser, url)
			Expect(err).ToNot(HaveOccurred())
			Expect(qrCode).To(HavePrefix("data:image/png;base64,"))
			Expect(len(qrCode)).To(BeNumerically(">", 100))
		})
	})
})
