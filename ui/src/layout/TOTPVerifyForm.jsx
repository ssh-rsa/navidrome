import React, { useState, useCallback } from 'react'
import { Field, Form } from 'react-final-form'
import Button from '@material-ui/core/Button'
import Card from '@material-ui/core/Card'
import CardActions from '@material-ui/core/CardActions'
import CircularProgress from '@material-ui/core/CircularProgress'
import TextField from '@material-ui/core/TextField'
import { makeStyles } from '@material-ui/core/styles'
import { useLogin, useNotify, useTranslate } from 'react-admin'
import Logo from '../icons/android-icon-192x192.png'

const useStyles = makeStyles(
  (theme) => ({
    main: {
      display: 'flex',
      flexDirection: 'column',
      minHeight: '100vh',
      alignItems: 'center',
      justifyContent: 'flex-start',
    },
    card: {
      minWidth: 300,
      marginTop: '6em',
      overflow: 'visible',
    },
    avatar: {
      margin: '1em',
      display: 'flex',
      justifyContent: 'center',
      marginTop: '-3em',
    },
    icon: {
      backgroundColor: 'transparent',
      width: '6.3em',
      height: '6.3em',
    },
    systemName: {
      marginTop: '1em',
      display: 'flex',
      justifyContent: 'center',
      color: '#3f51b5',
    },
    welcome: {
      marginTop: '1em',
      padding: '0 1em 1em 1em',
      display: 'flex',
      justifyContent: 'center',
      flexWrap: 'wrap',
      color: theme.palette.text.secondary,
    },
    form: {
      padding: '0 1em 1em 1em',
    },
    input: {
      marginTop: '1em',
    },
    actions: {
      padding: '0 1em 1em 1em',
    },
    button: {},
  }),
  { name: 'NDTOTPVerify' },
)

const renderInput = ({
  meta: { touched, error } = {},
  input: { ...inputProps },
  ...props
}) => (
  <TextField
    error={!!(touched && error)}
    helperText={touched && error}
    {...inputProps}
    {...props}
    fullWidth
  />
)

export const TOTPVerifyForm = ({ location, tempToken }) => {
  const [loading, setLoading] = useState(false)
  const translate = useTranslate()
  const notify = useNotify()
  const login = useLogin()
  const classes = useStyles()

  const handleSubmit = useCallback(
    (auth) => {
      setLoading(true)
      login(
        { totpCode: auth.totpCode, tempToken },
        location.state ? location.state.nextPathname : '/',
      ).catch((error) => {
        setLoading(false)
        notify(
          typeof error === 'string'
            ? error
            : typeof error === 'undefined' || !error.message
              ? 'ra.auth.sign_in_error'
              : error.message,
          'warning',
        )
      })
    },
    [login, notify, setLoading, location, tempToken],
  )

  const validate = useCallback(
    (values) => {
      const errors = {}
      if (!values.totpCode) {
        errors.totpCode = translate('ra.validation.required')
      } else if (!/^\d{6}$/.test(values.totpCode)) {
        errors.totpCode = translate('resources.user.validations.invalidTOTPCode')
      }
      return errors
    },
    [translate],
  )

  return (
    <Form
      onSubmit={handleSubmit}
      validate={validate}
      render={({ handleSubmit }) => (
        <form onSubmit={handleSubmit} noValidate>
          <div className={classes.main}>
            <Card className={classes.card}>
              <div className={classes.avatar}>
                <img src={Logo} className={classes.icon} alt={'logo'} />
              </div>
              <div className={classes.systemName}>Navidrome</div>
              <div className={classes.welcome}>
                {translate('ra.auth.totpVerifyMessage')}
              </div>
              <div className={classes.form}>
                <div className={classes.input}>
                  <Field
                    autoFocus
                    name="totpCode"
                    component={renderInput}
                    label={translate('ra.auth.verificationCode')}
                    disabled={loading}
                    inputProps={{ 
                      maxLength: 6,
                      pattern: '[0-9]*',
                      inputMode: 'numeric'
                    }}
                  />
                </div>
              </div>
              <CardActions className={classes.actions}>
                <Button
                  variant="contained"
                  type="submit"
                  color="primary"
                  disabled={loading}
                  className={classes.button}
                  fullWidth
                >
                  {loading && <CircularProgress size={25} thickness={2} />}
                  {translate('ra.auth.verify')}
                </Button>
              </CardActions>
            </Card>
          </div>
        </form>
      )}
    />
  )
}
