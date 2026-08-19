import { useState } from 'preact/hooks';

import { messages } from '../i18n/messages';
import {
  Archive,
  KeyRound,
  Lock,
  LogOut,
  FileUp,
  FileDown,
  Settings,
  SlidersHorizontal,
} from '../ui/icons/interface-icons';
import { Button, Menu, Tooltip, useToast } from '../ui/primitives';
import { logout } from './auth-api';
import { LoginDialog } from './login-dialog';
import { PasswordDialog } from './password-dialog';

type AuthenticationControlsProps = {
  administrator: boolean;
  compact?: boolean;
  onSessionChanged: () => void;
  onOpenSiteSettings?: () => void;
  onOpenBookmarkImport?: () => void;
  onOpenBookmarkExport?: () => void;
  onOpenBackups?: () => void;
  loginHref?: string;
};

export function AuthenticationControls({
  administrator,
  compact = false,
  onSessionChanged,
  onOpenSiteSettings,
  onOpenBookmarkImport,
  onOpenBookmarkExport,
  onOpenBackups,
  loginHref,
}: AuthenticationControlsProps) {
  const [loginOpen, setLoginOpen] = useState(false);
  const [passwordOpen, setPasswordOpen] = useState(false);
  const [loggingOut, setLoggingOut] = useState(false);
  const toast = useToast();

  const finishLogin = () => {
    toast.notify({
      tone: 'success',
      title: messages.auth.loginSucceeded,
      message: messages.auth.adminModeEnabled,
    });
    onSessionChanged();
  };

  const finishPasswordChange = () => {
    toast.notify({
      tone: 'success',
      title: messages.auth.passwordChanged,
      message: messages.auth.loginAgain,
    });
    onSessionChanged();
  };

  const performLogout = async () => {
    if (loggingOut) return;
    setLoggingOut(true);
    try {
      await logout();
      toast.notify({
        tone: 'success',
        title: messages.auth.loggedOut,
        message: messages.auth.publicModeEnabled,
      });
      onSessionChanged();
    } catch {
      toast.notify({
        tone: 'error',
        title: messages.auth.logoutFailedTitle,
        message: messages.auth.logoutFailed,
      });
    } finally {
      setLoggingOut(false);
    }
  };

  return (
    <>
      <div class={`auth-controls${compact ? ' auth-controls--compact' : ''}`}>
        {administrator ? (
          <Menu
            disabled={loggingOut}
            label={compact ? messages.auth.manageShort : messages.auth.manage}
            triggerIcon={Settings}
            items={[
              ...(onOpenBackups
                ? [
                    {
                      id: 'backups',
                      label: messages.backups.menu,
                      icon: Archive,
                      onSelect: onOpenBackups,
                    },
                  ]
                : []),
              ...(onOpenBookmarkImport
                ? [
                    {
                      id: 'import',
                      label: messages.imports.menu,
                      icon: FileUp,
                      onSelect: onOpenBookmarkImport,
                    },
                  ]
                : []),
              ...(onOpenBookmarkExport
                ? [
                    {
                      id: 'export',
                      label: messages.exports.menu,
                      icon: FileDown,
                      onSelect: onOpenBookmarkExport,
                    },
                  ]
                : []),
              ...(onOpenSiteSettings
                ? [
                    {
                      id: 'site',
                      label: messages.settings.menu,
                      icon: SlidersHorizontal,
                      onSelect: onOpenSiteSettings,
                    },
                  ]
                : []),
              {
                id: 'password',
                label: messages.auth.changePassword,
                icon: KeyRound,
                onSelect: () => setPasswordOpen(true),
              },
              {
                id: 'logout',
                label: messages.auth.logout,
                icon: LogOut,
                danger: true,
                onSelect: () => void performLogout(),
              },
            ]}
          />
        ) : loginHref ? (
          compact ? (
            <Tooltip content={messages.auth.login}>
              <Button
                variant="ghost"
                icon={Lock}
                aria-label={messages.auth.login}
                onClick={() => window.location.assign(loginHref)}
              />
            </Tooltip>
          ) : (
            <Button
              variant="ghost"
              icon={Lock}
              onClick={() => window.location.assign(loginHref)}
            >
              {messages.auth.login}
            </Button>
          )
        ) : compact ? (
          <Tooltip content={messages.auth.login}>
            <Button
              variant="ghost"
              icon={Lock}
              aria-label={messages.auth.login}
              onClick={() => setLoginOpen(true)}
            />
          </Tooltip>
        ) : (
          <Button icon={Lock} onClick={() => setLoginOpen(true)}>
            {messages.auth.login}
          </Button>
        )}
      </div>
      <LoginDialog
        open={loginOpen}
        onClose={() => setLoginOpen(false)}
        onSuccess={finishLogin}
      />
      <PasswordDialog
        open={passwordOpen}
        onClose={() => setPasswordOpen(false)}
        onSuccess={finishPasswordChange}
      />
    </>
  );
}
