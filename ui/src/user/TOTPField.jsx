import React, { useState } from 'react'
import { useTranslate, useNotify, useRefresh } from 'react-admin'
import { Box, Button, Typography, Chip } from '@material-ui/core'
import { makeStyles } from '@material-ui/core/styles'
import SecurityIcon from '@material-ui/icons/Security'
import TOTPSetupDialog from '../dialogs/TOTPSetupDialog'
import { httpClient } from '../dataProvider'

const useStyles = makeStyles((theme) => ({
  container: {
    marginTop: theme.spacing(2),
    marginBottom: theme.spacing(2),
  },
  statusChip: {
    marginLeft: theme.spacing(1),
  },
}))

export const TOTPField = ({ record, userId }) => {
  const translate = useTranslate()
  const notify = useNotify()
  const refresh = useRefresh()
  const classes = useStyles()
  const [setupDialogOpen, setSetupDialogOpen] = useState(false)
  const [loading, setLoading] = useState(false)

  const totpEnabled = record?.totpEnabled || false

  const handleDisableTOTP = async () => {
    if (!window.confirm(translate('resources.user.messages.confirmDisableTOTP'))) {
      return
    }

    setLoading(true)
    try {
      await httpClient(`/api/user/${userId}/totp/disable`, {
        method: 'POST',
        body: JSON.stringify({}),
      })
      notify(translate('resources.user.notifications.totpDisabled'), 'success')
      refresh()
    } catch (err) {
      notify(translate('resources.user.notifications.totpDisableFailed'), 'error')
    } finally {
      setLoading(false)
    }
  }

  const handleSetupSuccess = () => {
    refresh()
  }

  return (
    <Box className={classes.container}>
      <Typography variant="subtitle2" gutterBottom>
        <SecurityIcon fontSize="small" style={{ verticalAlign: 'middle', marginRight: 8 }} />
        {translate('resources.user.fields.twoFactorAuth')}
        <Chip
          className={classes.statusChip}
          label={
            totpEnabled
              ? translate('resources.user.status.enabled')
              : translate('resources.user.status.disabled')
          }
          color={totpEnabled ? 'primary' : 'default'}
          size="small"
        />
      </Typography>
      <Typography variant="body2" color="textSecondary" paragraph>
        {translate('resources.user.messages.totpDescription')}
      </Typography>
      {totpEnabled ? (
        <Button
          variant="outlined"
          color="secondary"
          onClick={handleDisableTOTP}
          disabled={loading}
        >
          {translate('resources.user.actions.disableTOTP')}
        </Button>
      ) : (
        <Button
          variant="contained"
          color="primary"
          onClick={() => setSetupDialogOpen(true)}
        >
          {translate('resources.user.actions.setupTOTP')}
        </Button>
      )}

      <TOTPSetupDialog
        open={setupDialogOpen}
        onClose={() => setSetupDialogOpen(false)}
        userId={userId}
        onSuccess={handleSetupSuccess}
      />
    </Box>
  )
}
