import { useEffect, useState } from 'preact/hooks';

import { messages } from '../i18n/messages';
import { LinkLogo } from '../navigation/link-logo';
import type { NavigationCategory, NavigationLink } from '../navigation/types';
import { RefreshCw } from '../ui/icons/interface-icons';
import { Button, Drawer, useToast } from '../ui/primitives';
import { SortableList } from '../ui/sortable';
import { refreshAllLinks, reorderLinks } from './link-api';

type Props = {
  categories: readonly NavigationCategory[];
  initialCategoryID: string;
  onClose: () => void;
  onSaved: () => void;
  open: boolean;
};

export function LinkOrderManager({
  categories,
  initialCategoryID,
  onClose,
  onSaved,
  open,
}: Props) {
  const [categoryID, setCategoryID] = useState(
    initialCategoryID || categories[0]?.id || '',
  );
  const category =
    categories.find((item) => item.id === categoryID) ?? categories[0];
  const [draft, setDraft] = useState<NavigationLink[]>([
    ...(category?.links ?? []),
  ]);
  const [saving, setSaving] = useState(false);
  const [refreshing, setRefreshing] = useState(false);
  const toast = useToast();
  useEffect(() => {
    if (open) {
      const selected =
        categories.find((item) => item.id === initialCategoryID) ??
        categories[0];
      setCategoryID(selected?.id ?? '');
      setDraft([...(selected?.links ?? [])]);
    }
  }, [categories, initialCategoryID, open]);
  const chooseCategory = (id: string) => {
    setCategoryID(id);
    setDraft([...(categories.find((item) => item.id === id)?.links ?? [])]);
  };
  const refreshAll = async () => {
    if (refreshing || saving) return;
    setRefreshing(true);
    try {
      const result = await refreshAllLinks();
      toast.notify({
        tone: result.failed ? 'info' : 'success',
        title: messages.links.refreshComplete,
        message: messages.links.refreshSummary(result.updated, result.failed),
      });
      onSaved();
    } catch {
      toast.notify({
        tone: 'error',
        title: messages.links.refreshFailed,
        message: messages.links.tryAgain,
      });
    } finally {
      setRefreshing(false);
    }
  };
  const orderChanged = draft.some(
    (link, index) => link.id !== category?.links[index]?.id,
  );
  const save = async () => {
    if (!category || !orderChanged || saving) return;
    setSaving(true);
    try {
      await reorderLinks(
        category.id,
        draft.map((link) => link.id),
      );
      toast.notify({
        tone: 'success',
        title: messages.links.orderSaved,
        message: messages.links.orderSavedDescription,
      });
      onSaved();
    } catch {
      toast.notify({
        tone: 'error',
        title: messages.links.orderFailed,
        message: messages.links.tryAgain,
      });
    } finally {
      setSaving(false);
    }
  };
  return (
    <Drawer
      open={open}
      title={messages.links.manage}
      onClose={saving || refreshing ? () => undefined : onClose}
    >
      <div class="link-order-manager">
        <label class="link-form__field">
          <span>{messages.links.category}</span>
          <select
            value={category?.id ?? ''}
            disabled={saving}
            onChange={(event) => chooseCategory(event.currentTarget.value)}
          >
            {categories.map((item) => (
              <option key={item.id} value={item.id}>
                {item.name}
              </option>
            ))}
          </select>
        </label>
        <div class="link-order-manager__refresh">
          <span>
            <strong>{messages.links.refreshAll}</strong>
            <small>{messages.links.refreshAllDescription}</small>
          </span>
          <Button
            icon={RefreshCw}
            loading={refreshing}
            disabled={saving}
            onClick={() => void refreshAll()}
          >
            {messages.links.refresh}
          </Button>
        </div>
        {draft.length ? (
          <SortableList
            items={draft}
            disabled={saving}
            getId={(link) => link.id}
            getLabel={(link) => link.name}
            onReorder={setDraft}
            renderItem={(link) => (
              <div class="link-order-manager__row">
                <LinkLogo
                  label={link.name}
                  text={link.logoText}
                  tone={link.logoTone}
                  url={link.logoURL}
                />
                <span>
                  <strong>{link.name}</strong>
                  <small>{link.url}</small>
                </span>
              </div>
            )}
          />
        ) : (
          <p class="link-order-manager__empty">{messages.links.empty}</p>
        )}
        <div class="link-form__actions">
          <span />
          <span>
            <Button disabled={saving} onClick={onClose}>
              {messages.links.close}
            </Button>
            <Button
              variant="primary"
              loading={saving}
              disabled={!orderChanged}
              onClick={() => void save()}
            >
              {messages.links.saveOrder}
            </Button>
          </span>
        </div>
      </div>
    </Drawer>
  );
}
