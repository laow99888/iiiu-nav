import { NavigationPage } from './navigation/navigation-page';
import { useNavigation } from './navigation/use-navigation';

export function App() {
  const navigation = useNavigation();
  return (
    <NavigationPage
      categories={navigation.snapshot?.categories ?? []}
      administrator={navigation.snapshot?.administrator ?? false}
      searchEngines={navigation.snapshot?.searchEngines}
      site={navigation.snapshot?.site}
      status={navigation.status}
      onRetry={navigation.retry}
      onSessionChanged={navigation.retry}
    />
  );
}
