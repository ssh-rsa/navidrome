package nativeapi

import (
	"encoding/json"
	"net/http"

	"github.com/deluan/rest"
	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
)

func (api *Router) addTOTPRoute(r chi.Router, totpSvc auth.TOTPService) {
	r.Route("/user/{id}/totp", func(r chi.Router) {
		r.Post("/setup", setupTOTP(api.ds, totpSvc))
		r.Post("/enable", enableTOTP(api.ds, totpSvc))
		r.Post("/disable", disableTOTP(api.ds))
	})
}

// setupTOTP generates a new TOTP secret and QR code for a user
func setupTOTP(ds model.DataStore, totpSvc auth.TOTPService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		userID := chi.URLParam(r, "id")

		// Get logged-in user
		loggedUser, ok := request.UserFrom(ctx)
		if !ok {
			_ = rest.RespondWithError(w, http.StatusUnauthorized, "Not authenticated")
			return
		}

		// Users can only setup TOTP for themselves, unless they're admin
		if !loggedUser.IsAdmin && loggedUser.ID != userID {
			_ = rest.RespondWithError(w, http.StatusForbidden, "Access denied")
			return
		}

		// Get the target user
		user, err := ds.User(ctx).Get(userID)
		if err != nil {
			log.Error(ctx, "Failed to get user", "userID", userID, err)
			_ = rest.RespondWithError(w, http.StatusNotFound, "User not found")
			return
		}

		// Generate TOTP secret and URL
		secret, url, err := totpSvc.GenerateSecret(ctx, user)
		if err != nil {
			log.Error(ctx, "Failed to generate TOTP secret", err)
			_ = rest.RespondWithError(w, http.StatusInternalServerError, "Failed to generate TOTP secret")
			return
		}

		// Generate QR code
		qrCode, err := totpSvc.GenerateQRCode(ctx, user, url)
		if err != nil {
			log.Error(ctx, "Failed to generate QR code", err)
			_ = rest.RespondWithError(w, http.StatusInternalServerError, "Failed to generate QR code")
			return
		}

		response := map[string]interface{}{
			"secret": secret,
			"qrCode": qrCode,
		}

		_ = rest.RespondWithJSON(w, http.StatusOK, response)
	}
}

// enableTOTP enables TOTP for a user after verifying a code
func enableTOTP(ds model.DataStore, totpSvc auth.TOTPService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		userID := chi.URLParam(r, "id")

		var data struct {
			Secret string `json:"secret"`
			Code   string `json:"code"`
		}

		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&data); err != nil {
			log.Error(ctx, "Failed to decode request body", err)
			_ = rest.RespondWithError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		// Get logged-in user
		loggedUser, ok := request.UserFrom(ctx)
		if !ok {
			_ = rest.RespondWithError(w, http.StatusUnauthorized, "Not authenticated")
			return
		}

		// Users can only enable TOTP for themselves, unless they're admin
		if !loggedUser.IsAdmin && loggedUser.ID != userID {
			_ = rest.RespondWithError(w, http.StatusForbidden, "Access denied")
			return
		}

		// Verify the TOTP code
		if !totpSvc.ValidateCode(ctx, data.Secret, data.Code) {
			log.Warn(ctx, "Invalid TOTP code during enable", "userID", userID)
			_ = rest.RespondWithError(w, http.StatusBadRequest, "Invalid verification code")
			return
		}

		// Get the user and update TOTP settings
		user, err := ds.User(ctx).Get(userID)
		if err != nil {
			log.Error(ctx, "Failed to get user", "userID", userID, err)
			_ = rest.RespondWithError(w, http.StatusNotFound, "User not found")
			return
		}

		user.TOTPSecret = data.Secret
		user.TOTPEnabled = true

		err = ds.User(ctx).Put(user)
		if err != nil {
			log.Error(ctx, "Failed to update user", "userID", userID, err)
			_ = rest.RespondWithError(w, http.StatusInternalServerError, "Failed to enable TOTP")
			return
		}

		response := map[string]interface{}{
			"success":     true,
			"totpEnabled": true,
		}

		_ = rest.RespondWithJSON(w, http.StatusOK, response)
	}
}

// disableTOTP disables TOTP for a user
func disableTOTP(ds model.DataStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		userID := chi.URLParam(r, "id")

		// Get logged-in user
		loggedUser, ok := request.UserFrom(ctx)
		if !ok {
			_ = rest.RespondWithError(w, http.StatusUnauthorized, "Not authenticated")
			return
		}

		// Users can only disable TOTP for themselves, unless they're admin
		if !loggedUser.IsAdmin && loggedUser.ID != userID {
			_ = rest.RespondWithError(w, http.StatusForbidden, "Access denied")
			return
		}

		// Get the user and update TOTP settings
		user, err := ds.User(ctx).Get(userID)
		if err != nil {
			log.Error(ctx, "Failed to get user", "userID", userID, err)
			_ = rest.RespondWithError(w, http.StatusNotFound, "User not found")
			return
		}

		user.TOTPSecret = ""
		user.TOTPEnabled = false

		err = ds.User(ctx).Put(user)
		if err != nil {
			log.Error(ctx, "Failed to update user", "userID", userID, err)
			_ = rest.RespondWithError(w, http.StatusInternalServerError, "Failed to disable TOTP")
			return
		}

		response := map[string]interface{}{
			"success":     true,
			"totpEnabled": false,
		}

		_ = rest.RespondWithJSON(w, http.StatusOK, response)
	}
}
