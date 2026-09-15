import { useEffect } from 'preact/hooks';

import { NavigationPage } from './navigation/navigation-page';
import { useNavigation } from './navigation/use-navigation';
import { defaultSiteSettings } from './navigation/types';
import { AdminLoginPage } from './admin/admin-login-page';
import { AdminLoadingPage } from './admin/admin-loading-page';
import { AdminPage } from './admin/admin-page';

export function App() {
  const adminRoute = window.location.pathname.startsWith('/admin');
  const adminLoginRoute = window.location.pathname === '/admin/login';
  const navigation = useNavigation(adminRoute ? 'all' : 'public');
  const accentColor =
    navigation.snapshot?.site?.accentColor ?? defaultSiteSettings.accentColor;
  useEffect(() => {
    // The accent drives :root-level derived tokens, so it must be set on the
    // document root where those tokens are declared — a subtree override
    // would not re-derive them and would also miss body-level dialog portals.
    document.documentElement.style.setProperty('--site-accent', accentColor);
  }, [accentColor]);
  if (adminRoute && navigation.status === 'ready') {
    if (!navigation.snapshot?.administrator) {
      return <AdminLoginPage site={navigation.snapshot.site} />;
    }
    return (
      <AdminPage
        categories={navigation.snapshot.categories}
        onRetry={navigation.retry}
        onSessionChanged={navigation.retry}
        searchEngines={navigation.snapshot.searchEngines}
        site={navigation.snapshot.site}
      />
    );
  }
  if (adminRoute) {
    if (navigation.status === 'loading' && !adminLoginRoute) {
      return <AdminLoadingPage />;
    }
    return (
      <AdminLoginPage
        site={undefined}
        loading={navigation.status === 'loading'}
      />
    );
  }
  return (
    <NavigationPage
      categories={navigation.snapshot?.categories ?? []}
      administrator={false}
      searchEngines={navigation.snapshot?.searchEngines}
      site={navigation.snapshot?.site}
      status={navigation.status}
      onRetry={navigation.retry}
      onSessionChanged={navigation.retry}
      loginHref={navigation.snapshot?.administrator ? '/admin' : '/admin/login'}
    />
  );
}
