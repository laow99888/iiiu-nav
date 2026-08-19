import { useEffect, useState } from 'preact/hooks';

import { messages } from '../i18n/messages';
import type { NavigationCategory } from '../navigation/types';
import { CategoryIcon } from '../ui/icons/category-icons';
import { Lock, Plus, Settings, Trash2 } from '../ui/icons/interface-icons';
import { Button, Drawer, Tooltip, useToast } from '../ui/primitives';
import { SortableList } from '../ui/sortable';
import { reorderCategories } from './category-api';
import { CategoryDeleteDialog } from './category-delete-dialog';
import { CategoryForm } from './category-form';

type Props = {
  categories: readonly NavigationCategory[];
  initialAction?: CategoryManagerAction | null;
  onClose: () => void;
  onSaved: () => void;
  open: boolean;
};
export type CategoryManagerAction =
  | { type: 'create' }
  | { type: 'edit'; categoryID: string }
  | { type: 'delete'; categoryID: string };
type View =
  | { type: 'list' }
  | { type: 'create' }
  | { type: 'edit'; category: NavigationCategory };

export function CategoryManager({
  categories,
  initialAction,
  onClose,
  onSaved,
  open,
}: Props) {
  const [view, setView] = useState<View>(() =>
    initialView(categories, initialAction),
  );
  const [draft, setDraft] = useState<NavigationCategory[]>([...categories]);
  const [deleting, setDeleting] = useState<NavigationCategory | null>(() =>
    initialDeletingCategory(categories, initialAction),
  );
  const [savingOrder, setSavingOrder] = useState(false);
  const toast = useToast();
  useEffect(() => {
    if (open) {
      setDraft([...categories]);
      const categoryID =
        initialAction && 'categoryID' in initialAction
          ? initialAction.categoryID
          : '';
      const category = categoryID
        ? categories.find((item) => item.id === categoryID)
        : undefined;
      if (initialAction?.type === 'create') {
        setView({ type: 'create' });
        setDeleting(null);
      } else if (initialAction?.type === 'edit' && category) {
        setView({ type: 'edit', category });
        setDeleting(null);
      } else if (initialAction?.type === 'delete' && category) {
        setView({ type: 'list' });
        setDeleting(category);
      } else {
        setView({ type: 'list' });
        setDeleting(null);
      }
    }
  }, [categories, initialAction, open]);
  const close = () => {
    if (!savingOrder) {
      setView({ type: 'list' });
      onClose();
    }
  };
  const finishMutation = () => {
    setDeleting(null);
    onSaved();
    onClose();
  };
  const orderChanged = draft.some(
    (category, index) => category.id !== categories[index]?.id,
  );
  const saveOrder = async () => {
    if (!orderChanged || savingOrder) return;
    setSavingOrder(true);
    try {
      await reorderCategories(draft.map((category) => category.id));
      toast.notify({
        tone: 'success',
        title: messages.categories.orderSaved,
        message: messages.categories.orderSavedDescription,
      });
      finishMutation();
    } catch {
      toast.notify({
        tone: 'error',
        title: messages.categories.orderFailed,
        message: messages.categories.tryAgain,
      });
    } finally {
      setSavingOrder(false);
    }
  };
  const formCategory = view.type === 'edit' ? view.category : null;
  return (
    <>
      <Drawer
        open={open}
        title={
          view.type === 'list'
            ? messages.categories.manage
            : formCategory
              ? messages.categories.edit
              : messages.categories.create
        }
        onClose={close}
      >
        {view.type === 'list' ? (
          <div class="category-manager">
            <div class="category-manager__intro">
              <p>{messages.categories.manageDescription}</p>
              <Button icon={Plus} onClick={() => setView({ type: 'create' })}>
                {messages.categories.add}
              </Button>
            </div>
            {draft.length > 0 ? (
              <SortableList
                items={draft}
                disabled={savingOrder}
                getId={(category) => category.id}
                getLabel={(category) => category.name}
                onReorder={setDraft}
                renderItem={(category) => (
                  <div class="category-manager__row">
                    <CategoryIcon name={category.icon} />
                    <span>
                      <strong>{category.name}</strong>
                      <small>
                        {category.links.length} {messages.categories.links}
                        {category.visibility === 'private' ? (
                          <Lock
                            aria-label={messages.navigation.privateCategory}
                          />
                        ) : null}
                      </small>
                    </span>
                    <Tooltip
                      content={messages.categories.editNamed(category.name)}
                    >
                      <Button
                        variant="ghost"
                        size="small"
                        icon={Settings}
                        aria-label={messages.categories.editNamed(
                          category.name,
                        )}
                        onClick={() => setView({ type: 'edit', category })}
                      />
                    </Tooltip>
                    <Tooltip
                      content={messages.categories.deleteNamed(category.name)}
                    >
                      <Button
                        variant="ghost"
                        size="small"
                        icon={Trash2}
                        aria-label={messages.categories.deleteNamed(
                          category.name,
                        )}
                        onClick={() => setDeleting(category)}
                      />
                    </Tooltip>
                  </div>
                )}
              />
            ) : (
              <p class="category-manager__empty">{messages.categories.empty}</p>
            )}
            <div class="category-manager__actions">
              <Button onClick={close}>{messages.categories.close}</Button>
              <Button
                variant="primary"
                loading={savingOrder}
                disabled={!orderChanged}
                onClick={() => void saveOrder()}
              >
                {messages.categories.saveOrder}
              </Button>
            </div>
          </div>
        ) : (
          <CategoryForm
            key={formCategory?.id ?? 'create'}
            category={formCategory}
            onCancel={() => setView({ type: 'list' })}
            onSaved={finishMutation}
          />
        )}
      </Drawer>
      <CategoryDeleteDialog
        category={deleting}
        categories={categories}
        onClose={() => setDeleting(null)}
        onDeleted={finishMutation}
      />
    </>
  );
}

function initialView(
  categories: readonly NavigationCategory[],
  action?: CategoryManagerAction | null,
): View {
  if (action?.type === 'create') return { type: 'create' };
  if (action?.type === 'edit') {
    const category = categories.find((item) => item.id === action.categoryID);
    if (category) return { type: 'edit', category };
  }
  return { type: 'list' };
}

function initialDeletingCategory(
  categories: readonly NavigationCategory[],
  action?: CategoryManagerAction | null,
) {
  if (action?.type !== 'delete') return null;
  return categories.find((item) => item.id === action.categoryID) ?? null;
}
