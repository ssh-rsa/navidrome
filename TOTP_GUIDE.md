# TOTP Multi-Factor Authentication

Navidrome now supports TOTP (Time-based One-Time Password) multi-factor authentication for enhanced account security.

## User Guide

### Setting Up TOTP

1. Log in to your Navidrome account
2. Go to your user profile (Settings → User Settings)
3. Scroll down to the "Two-Factor Authentication" section
4. Click the **"Setup TOTP"** button
5. A QR code will be displayed
6. Scan the QR code with your authenticator app (e.g., Google Authenticator, Authy, Microsoft Authenticator)
7. Enter the 6-digit verification code from your authenticator app
8. Click **"Enable TOTP"**

### Logging In with TOTP

Once TOTP is enabled:

1. Enter your username and password as usual
2. You'll be prompted to enter a verification code
3. Open your authenticator app and enter the current 6-digit code
4. Click **"Verify"**

### Disabling TOTP

1. Go to your user profile
2. In the "Two-Factor Authentication" section, click **"Disable TOTP"**
3. Confirm the action

**Warning:** Make sure you have access to your authenticator app before disabling TOTP, or you may lock yourself out of your account.

## API Documentation

### TOTP Setup Endpoint

**POST** `/api/user/:id/totp/setup`

Generates a new TOTP secret and QR code for the specified user.

**Authentication:** Required (user can only setup TOTP for themselves, unless admin)

**Response:**
```json
{
  "secret": "JBSWY3DPEHPK3PXP",
  "qrCode": "data:image/png;base64,iVBORw0KGgoAAAANS..."
}
```

### Enable TOTP Endpoint

**POST** `/api/user/:id/totp/enable`

Enables TOTP for the user after verifying a code.

**Authentication:** Required (user can only enable TOTP for themselves, unless admin)

**Request Body:**
```json
{
  "secret": "JBSWY3DPEHPK3PXP",
  "code": "123456"
}
```

**Response:**
```json
{
  "success": true,
  "totpEnabled": true
}
```

### Disable TOTP Endpoint

**POST** `/api/user/:id/totp/disable`

Disables TOTP for the user.

**Authentication:** Required (user can only disable TOTP for themselves, unless admin)

**Response:**
```json
{
  "success": true,
  "totpEnabled": false
}
```

### TOTP Verification Endpoint

**POST** `/auth/totp/verify`

Verifies a TOTP code during login and completes the authentication.

**Authentication:** Requires temporary token from initial login

**Request Body:**
```json
{
  "tempToken": "eyJhbGciOiJIUzI1NiIs...",
  "code": "123456"
}
```

**Response:**
```json
{
  "id": "user-id",
  "name": "User Name",
  "username": "username",
  "isAdmin": false,
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "subsonicSalt": "abc123",
  "subsonicToken": "def456"
}
```

## Authentication Flow

### Without TOTP

1. **POST** `/auth/login` with username and password
2. Receive authentication token
3. Access protected resources

### With TOTP Enabled

1. **POST** `/auth/login` with username and password
2. Receive response with `totpRequired: true` and `tempToken`
3. **POST** `/auth/totp/verify` with tempToken and TOTP code
4. Receive final authentication token
5. Access protected resources

## Database Schema

The TOTP feature adds two new columns to the `user` table:

- `totp_secret` (VARCHAR(255)): Encrypted TOTP secret key
- `totp_enabled` (BOOL): Whether TOTP is enabled for the user

### Database Migration

**Important:** The database migration runs automatically when you upgrade Navidrome. No manual intervention is required.

#### How It Works

When you start Navidrome after upgrading to a version with TOTP support:

1. Navidrome automatically detects pending migrations on startup
2. The migration `20260101213142_add_totp_fields.sql` is applied automatically
3. The new `totp_secret` and `totp_enabled` columns are added to the `user` table
4. Existing users are unaffected - both fields default to empty/false
5. You'll see a log message: `"Upgrading DB Schema to latest version"`

#### Migration File Location

The migration file is located at:
```
db/migrations/20260101213142_add_totp_fields.sql
```

#### Manual Migration (Advanced)

If you need to manually verify or run migrations, you can use the `goose` tool:

```bash
# Check migration status
goose -dir db/migrations sqlite3 /path/to/navidrome.db status

# Apply pending migrations
goose -dir db/migrations sqlite3 /path/to/navidrome.db up

# Rollback the TOTP migration (if needed)
goose -dir db/migrations sqlite3 /path/to/navidrome.db down
```

**Note:** Manual migration management is rarely needed as Navidrome handles this automatically. Only use these commands if you're troubleshooting or have specific requirements.

#### Backup Recommendation

Before upgrading to a version with database schema changes, it's recommended to:

1. Stop Navidrome
2. Backup your database file (usually `navidrome.db`)
3. Start the upgraded Navidrome version
4. Verify everything works correctly

You can find your database file location in your configuration or server logs.

## Security Considerations

- TOTP secrets are stored encrypted in the database
- Temporary tokens for TOTP verification expire after 5 minutes
- TOTP codes are 6 digits and follow the standard TOTP algorithm (RFC 6238)
- Users should be encouraged to store backup codes or recovery methods in case they lose access to their authenticator app

## Compatible Authenticator Apps

- Google Authenticator (iOS, Android)
- Microsoft Authenticator (iOS, Android)
- Authy (iOS, Android, Desktop)
- 1Password
- LastPass Authenticator
- Any other TOTP-compatible authenticator app

## Implementation Details

### Backend

- TOTP service: `core/auth/totp.go`
- Uses `github.com/pquerna/otp` library for TOTP generation and validation
- QR codes generated as base64-encoded PNG images
- Migration: `db/migrations/20260101213142_add_totp_fields.sql`

#### Detailed Code Breakdown: `core/auth/totp.go`

**Package and Imports (Lines 1-14)**
```go
package auth
```
- Part of the `auth` package which handles authentication logic

**Imports:**
- `bytes` - For buffering QR code image data
- `context` - For passing request context through function calls
- `encoding/base64` - To encode QR code images as base64 strings
- `errors` - For creating custom error types
- `image/png` - For encoding QR codes as PNG images
- `github.com/navidrome/navidrome/log` - Application logging
- `github.com/navidrome/navidrome/model` - Data models (User struct)
- `github.com/pquerna/otp` - OTP library for key generation
- `github.com/pquerna/otp/totp` - TOTP-specific functionality

**Error Definitions (Lines 16-18)**
```go
var (
    ErrInvalidTOTPCode = errors.New("invalid TOTP code")
)
```
- Defines a custom error for invalid TOTP codes
- Can be checked with `errors.Is()` for specific error handling

**Service Interface (Lines 20-24)**
```go
type TOTPService interface {
    GenerateSecret(ctx context.Context, user *model.User) (string, string, error)
    ValidateCode(ctx context.Context, secret string, code string) bool
    GenerateQRCode(ctx context.Context, user *model.User, secret string) (string, error)
}
```
- Defines the contract for TOTP operations
- `GenerateSecret` - Returns (secret, URL, error) for TOTP setup
- `ValidateCode` - Returns bool indicating if the provided code is valid
- `GenerateQRCode` - Returns base64-encoded QR code image or error

**Service Implementation (Lines 26-30)**
```go
type totpService struct{}

func NewTOTPService() TOTPService {
    return &totpService{}
}
```
- `totpService` is a stateless implementation of `TOTPService`
- `NewTOTPService` is the constructor function used for dependency injection
- Returns interface type to allow future alternative implementations

**GenerateSecret Method (Lines 32-44)**
```go
func (s *totpService) GenerateSecret(ctx context.Context, user *model.User) (string, string, error)
```
Line-by-line breakdown:
- **Line 33**: Function declaration accepting context and user
- **Line 34-37**: Call `totp.Generate()` with configuration:
  - `Issuer: "Navidrome"` - Shows as issuer in authenticator apps
  - `AccountName: user.UserName` - Shows username in authenticator apps
- **Line 38-41**: Error handling - logs failure with user context and returns error
- **Line 43**: Returns three values:
  - `key.Secret()` - The base32-encoded secret string to store in database
  - `key.URL()` - The `otpauth://` URL for QR code generation
  - `nil` - No error occurred

**ValidateCode Method (Lines 46-53)**
```go
func (s *totpService) ValidateCode(ctx context.Context, secret string, code string) bool
```
Line-by-line breakdown:
- **Line 47**: Function declaration accepting context, secret, and user-provided code
- **Line 48**: Call `totp.Validate()` to verify the code against the secret
  - Uses current time window (±1 period for clock skew tolerance)
  - Returns boolean indicating validity
- **Line 49-51**: If invalid, log a warning (helps detect brute force attempts)
- **Line 52**: Return the validation result

**GenerateQRCode Method (Lines 55-81)**
```go
func (s *totpService) GenerateQRCode(ctx context.Context, user *model.User, secret string) (string, error)
```
Line-by-line breakdown:
- **Line 56**: Function declaration accepting context, user, and secret URL
- **Line 57-61**: Parse the `otpauth://` URL into a Key object
  - **Line 57**: Call `otp.NewKeyFromURL()` to parse the URL
  - **Line 58-60**: Error handling with logging
- **Line 64-68**: Generate QR code image
  - **Line 64**: Call `key.Image(200, 200)` to create 200x200 pixel QR code
  - **Line 65-67**: Error handling with logging
- **Line 71-76**: Encode QR code as PNG
  - **Line 71**: Create a buffer to hold PNG data
  - **Line 72**: Encode the image as PNG into the buffer
  - **Line 73-75**: Error handling with logging
- **Line 79-80**: Convert to base64 data URL
  - **Line 79**: Encode buffer bytes to base64 string
  - **Line 80**: Prepend `data:image/png;base64,` prefix for direct browser use
  - Returns the complete data URL that can be used in `<img>` tags

**Key Design Decisions:**
1. **Stateless Service** - No internal state, all data passed as parameters
2. **Context Propagation** - All methods accept context for cancellation and logging
3. **Error Logging** - All errors logged with context before returning
4. **Base64 Data URLs** - QR codes returned as data URLs for easy frontend integration
5. **Standard Compliance** - Uses RFC 6238 compliant TOTP implementation
6. **Time Window** - Default 30-second time window with ±1 period skew tolerance

### Frontend

- Setup dialog: `ui/src/dialogs/TOTPSetupDialog.jsx`
- User settings field: `ui/src/user/TOTPField.jsx`
- Login verification: `ui/src/layout/TOTPVerifyForm.jsx`
- AuthProvider integration: `ui/src/authProvider.js`

## Testing

Unit tests for the TOTP service can be run with:

```bash
go test -tags netgo ./core/auth
```

All TOTP-related tests should pass before deploying.
