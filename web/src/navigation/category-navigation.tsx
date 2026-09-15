import { messages } from '../i18n/messages';
import { CategoryIcon } from '../ui/icons/category-icons';
import { LayoutGrid, Lock } from '../ui/icons/interface-icons';
import type { NavigationCategory } from './types';

type CategoryNavigationProps = {
  activeId: string;
  categories: readonly NavigationCategory[];
  onSelect: (id: string) => void;
  variant: 'desktop' | 'mobile';
};

export function CategoryNavigation({
  activeId,
  categories,
  onSelect,
  variant,
}: CategoryNavigationProps) {
  const items = [
    {
      id: 'all',
      name: messages.navigation.allCategories,
      icon: null,
      private: false,
      count: categories.reduce(
        (sum, category) => sum + category.links.length,
        0,
      ),
    },
    ...categories.map((category) => ({
      id: category.id,
      name: category.name,
      icon: category.icon,
      private: category.visibility === 'private',
      count: category.links.length,
    })),
  ];

  return (
    <nav
      class={`category-nav category-nav--${variant}`}
      aria-label={
        variant === 'desktop'
          ? messages.navigation.desktopCategories
          : messages.navigation.mobileCategories
      }
    >
      {items.map((item) => {
        const active = activeId === item.id;
        return (
          <button
            key={item.id}
            type="button"
            class={`category-nav__item${active ? ' is-active' : ''}`}
            aria-current={active ? 'page' : undefined}
            onClick={() => onSelect(item.id)}
          >
            {item.id === 'all' ? (
              <LayoutGrid aria-hidden="true" />
            ) : item.icon ? (
              <CategoryIcon name={item.icon} />
            ) : (
              <span class="category-nav__placeholder" aria-hidden="true" />
            )}
            <span class="category-nav__name">
              <span>{item.name}</span>
              {item.private ? (
                <Lock
                  class="category-nav__private"
                  aria-label={messages.navigation.privateCategory}
                />
              ) : null}
            </span>
            <span
              class="category-nav__count"
              aria-label={messages.navigation.linkCount(item.count)}
            >
              {item.count}
            </span>
          </button>
        );
      })}
    </nav>
  );
}
