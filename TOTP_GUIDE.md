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
