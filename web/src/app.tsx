import { NavigationPage } from './navigation/navigation-page';
import { useNavigation } from './navigation/use-navigation';
import { AdminLoginPage } from './admin/admin-login-page';
import { AdminLoadingPage } from './admin/admin-loading-page';
import { AdminPage } from './admin/admin-page';

export function App() {
  const adminRoute = window.location.pathname.startsWith('/admin');
  const adminLoginRoute = window.location.pathname === '/admin/login';
  const navigation = useNavigation(adminRoute ? 'all' : 'public');
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
