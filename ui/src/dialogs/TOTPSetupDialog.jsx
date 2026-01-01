import React, { useState } from 'react'
import {
  Dialog,
  DialogActions,
  DialogContent,
  DialogContentText,
  Button,
  TextField,
  Box,
  Typography,
  CircularProgress,
} from '@material-ui/core'
import { useTranslate, useNotify, useDataProvider } from 'react-admin'
import { DialogTitle } from './DialogTitle'

const TOTPSetupDialog = ({ open, onClose, userId, onSuccess }) => {
  const translate = useTranslate()
  const notify = useNotify()
  const dataProvider = useDataProvider()
  const [step, setStep] = useState(1) // 1: Generate QR, 2: Verify code
  const [qrCode, setQrCode] = useState('')
  const [secret, setSecret] = useState('')
  const [code, setCode] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  const handleGenerateSecret = async () => {
    setLoading(true)
    setError('')
    try {
      const response = await dataProvider.create(`user/${userId}/totp/setup`, {
        data: {},
      })
      setSecret(response.data.secret)
      setQrCode(response.data.qrCode)
      setStep(2)
    } catch (err) {
      notify(translate('resources.user.notifications.totpSetupFailed'), 'error')
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  const handleEnableTOTP = async () => {
    if (!code || code.length !== 6) {
      setError(translate('resources.user.validations.invalidTOTPCode'))
      return
    }

    setLoading(true)
    setError('')
    try {
      await dataProvider.create(`user/${userId}/totp/enable`, {
        data: {
          secret: secret,
          code: code,
        },
      })
      notify(translate('resources.user.notifications.totpEnabled'), 'success')
      onSuccess && onSuccess()
      handleClose()
    } catch (err) {
      notify(translate('resources.user.notifications.totpEnableFailed'), 'error')
      setError(translate('resources.user.validations.invalidTOTPCode'))
    } finally {
      setLoading(false)
    }
  }

  const handleClose = () => {
    setStep(1)
    setQrCode('')
    setSecret('')
    setCode('')
    setError('')
    onClose()
  }

  return (
    <Dialog open={open} onClose={handleClose} maxWidth="sm" fullWidth>
      <DialogTitle onClose={handleClose}>
        {translate('resources.user.fields.setupTOTP')}
      </DialogTitle>
      <DialogContent>
        {step === 1 && (
          <>
            <DialogContentText>
              {translate('resources.user.messages.totpSetupInstructions')}
            </DialogContentText>
            {loading && (
              <Box display="flex" justifyContent="center" p={3}>
                <CircularProgress />
              </Box>
            )}
          </>
        )}

        {step === 2 && (
          <>
            <DialogContentText>
              {translate('resources.user.messages.scanQRCode')}
            </DialogContentText>
            {qrCode && (
              <Box display="flex" flexDirection="column" alignItems="center" mt={2}>
                <img
                  src={qrCode}
                  alt="TOTP QR Code"
                  style={{ width: 200, height: 200 }}
                />
                <Typography variant="caption" color="textSecondary" style={{ marginTop: 16 }}>
                  {translate('resources.user.messages.manualEntry')}: {secret}
                </Typography>
              </Box>
            )}
            <Box mt={3}>
              <TextField
                fullWidth
                label={translate('resources.user.fields.verificationCode')}
                value={code}
                onChange={(e) => setCode(e.target.value.replace(/\D/g, '').slice(0, 6))}
                placeholder="000000"
                error={!!error}
                helperText={error}
                disabled={loading}
                inputProps={{ maxLength: 6, pattern: '[0-9]*' }}
              />
            </Box>
          </>
        )}
      </DialogContent>
      <DialogActions>
        <Button onClick={handleClose} disabled={loading}>
          {translate('ra.action.cancel')}
        </Button>
        {step === 1 && (
          <Button
            onClick={handleGenerateSecret}
            color="primary"
            disabled={loading}
            variant="contained"
          >
            {translate('resources.user.actions.generateQRCode')}
          </Button>
        )}
        {step === 2 && (
          <Button
            onClick={handleEnableTOTP}
            color="primary"
            disabled={loading || code.length !== 6}
            variant="contained"
          >
            {translate('resources.user.actions.enableTOTP')}
          </Button>
        )}
      </DialogActions>
    </Dialog>
  )
}

export default TOTPSetupDialog
