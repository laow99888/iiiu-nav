import { messages } from '../i18n/messages';
import { ArrowUpRight, Pencil } from '../ui/icons/interface-icons';
import { Button, EmptyState, Tooltip } from '../ui/primitives';
import { CategoryIcon } from '../ui/icons/category-icons';
import { LinkLogo } from './link-logo';
import type { NavigationCategory, NavigationLink } from './types';

function LinkCard({
  link,
  onEdit,
}: {
  link: NavigationLink;
  onEdit?: () => void;
}) {
  return (
    <div class={`link-card-shell${onEdit ? ' is-editable' : ''}`}>
      <a
        class="link-card"
        href={link.url}
        target="_blank"
        rel="noopener noreferrer"
        aria-label={`${link.name}，${link.description}`}
      >
        <LinkLogo
          label={link.name}
          text={link.logoText}
          tone={link.logoTone}
          url={link.logoURL}
        />
        <span class="link-card__copy">
          <strong>{link.name}</strong>
          <span>{link.description}</span>
        </span>
        <ArrowUpRight class="link-card__external" aria-hidden="true" />
      </a>
      {onEdit ? (
        <Tooltip content={messages.links.editNamed(link.name)}>
          <Button
            class="link-card__edit"
            variant="ghost"
            size="small"
            icon={Pencil}
            aria-label={messages.links.editNamed(link.name)}
            onClick={onEdit}
          />
        </Tooltip>
      ) : null}
    </div>
  );
}

type LinkGridProps = {
  category: NavigationCategory | null;
  links: readonly NavigationLink[];
  onRetry?: () => void;
  onEdit?: (link: NavigationLink) => void;
  status?: 'ready' | 'loading' | 'error';
};

export function LinkGrid({
  category,
  links,
  onRetry,
  onEdit,
  status = 'ready',
}: LinkGridProps) {
  if (status === 'loading') {
    return (
      <EmptyState
        state="loading"
        title={messages.navigation.loadingTitle}
        description={messages.navigation.loadingDescription}
      />
    );
  }

  if (status === 'error') {
    return (
      <EmptyState
        state="error"
        title={messages.navigation.errorTitle}
        description={messages.navigation.errorDescription}
        action={
          onRetry ? (
            <Button onClick={onRetry}>{messages.navigation.retry}</Button>
          ) : null
        }
      />
    );
  }

  if (links.length === 0) {
    const isNavigationEmpty = category === null;
    return (
      <EmptyState
        icon={CategoryIcon}
        title={
          isNavigationEmpty
            ? messages.navigation.navigationEmptyTitle
            : messages.navigation.categoryEmptyTitle
        }
        description={
          isNavigationEmpty
            ? messages.navigation.navigationEmptyDescription
            : messages.navigation.categoryEmptyDescription
        }
      />
    );
  }

  return (
    <section
      class="link-grid"
      aria-label={
        category
          ? messages.navigation.categoryLinks(category.name)
          : messages.navigation.allLinks
      }
    >
      {links.map((link) => (
        <LinkCard
          key={link.id}
          link={link}
          onEdit={onEdit ? () => onEdit(link) : undefined}
        />
      ))}
    </section>
  );
}
