import { useEffect, useState } from 'preact/hooks';

import { AuthAPIError, login } from '../auth/auth-api';
import { messages } from '../i18n/messages';
import { Compass, LoaderCircle, Lock } from '../ui/icons/interface-icons';
import { Button, TextField } from '../ui/primitives';
import { SiteIdentity } from '../navigation/site-identity';
import type { SiteSettings } from '../navigation/types';

type Props = {
  loading?: boolean;
  site?: SiteSettings;
};

export function AdminLoginPage({ loading = false, site }: Props) {
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    document.title = site?.name
      ? `${messages.admin.loginTitle} · ${site.name}`
      : messages.admin.loginTitle;
  }, [site?.name]);

  const submit = async () => {
    if (!password || submitting || loading) return;
    setSubmitting(true);
    setError('');
    try {
      await login(password);
      window.location.assign('/admin');
    } catch (cause) {
      setError(loginErrorMessage(cause));
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <main class="admin-login-shell">
      <section class="admin-login-card" aria-labelledby="admin-login-title">
        <SiteIdentity name={site?.name ?? 'iiiu-nav'} logoUrl={site?.logoUrl} />
        <div class="admin-login-card__intro">
          <span class="admin-login-card__icon" aria-hidden="true">
            {loading ? (
              <LoaderCircle class="ui-button__spinner" />
            ) : (
              <Compass />
            )}
          </span>
          <p>{messages.admin.eyebrow}</p>
          <h1 id="admin-login-title">{messages.admin.loginTitle}</h1>
          <span>{messages.admin.loginDescription}</span>
        </div>
        <form
          class="admin-login-form"
          onSubmit={(event) => {
            event.preventDefault();
            void submit();
          }}
        >
          <TextField
            autoFocus
            required
            type="password"
            autoComplete="current-password"
            label={messages.auth.password}
            value={password}
            disabled={loading || submitting}
            error={error || undefined}
            onInput={(event) => setPassword(event.currentTarget.value)}
          />
          <Button
            type="submit"
            variant="primary"
            icon={Lock}
            loading={submitting || loading}
            disabled={!password || loading}
          >
            {messages.auth.login}
          </Button>
        </form>
        <a class="admin-login-card__back" href="/">
          {messages.admin.backToPublic}
        </a>
      </section>
    </main>
  );
}

function loginErrorMessage(cause: unknown) {
  if (cause instanceof AuthAPIError) {
    if (cause.code === 'invalid_credentials')
      return messages.auth.invalidCredentials;
    if (cause.code === 'login_rate_limited') {
      return cause.retryAfter
        ? messages.auth.rateLimitedSeconds(cause.retryAfter)
        : messages.auth.rateLimited;
    }
  }
  return messages.auth.loginFailed;
}
