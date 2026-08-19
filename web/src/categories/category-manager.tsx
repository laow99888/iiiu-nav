import { useEffect, useState } from 'preact/hooks';

import { messages } from '../i18n/messages';
import type { NavigationCategory } from '../navigation/types';
import { CategoryIcon } from '../ui/icons/category-icons';
import { Lock, Plus, Settings, Trash2 } from '../ui/icons/interface-icons';
import { Button, Dialog, Drawer, Tooltip, useToast } from '../ui/primitives';
import { SortableList } from '../ui/sortable';
import { deleteCategory, reorderCategories } from './category-api';
import { CategoryForm } from './category-form';

type Props = {
  categories: readonly NavigationCategory[];
  onClose: () => void;
  onSaved: () => void;
  open: boolean;
};
type View =
  | { type: 'list' }
  | { type: 'create' }
  | { type: 'edit'; category: NavigationCategory };

export function CategoryManager({ categories, onClose, onSaved, open }: Props) {
  const [view, setView] = useState<View>({ type: 'list' });
  const [draft, setDraft] = useState<NavigationCategory[]>([...categories]);
  const [deleting, setDeleting] = useState<NavigationCategory | null>(null);
  const [savingOrder, setSavingOrder] = useState(false);
  const toast = useToast();
  useEffect(() => {
    if (open) {
      setDraft([...categories]);
      setView({ type: 'list' });
    }
  }, [categories, open]);
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

type DeleteProps = {
  categories: readonly NavigationCategory[];
  category: NavigationCategory | null;
  onClose: () => void;
  onDeleted: () => void;
};

function CategoryDeleteDialog({
  categories,
  category,
  onClose,
  onDeleted,
}: DeleteProps) {
  const targets = categories.filter((item) => item.id !== category?.id);
  const hasLinks = (category?.links.length ?? 0) > 0;
  const [mode, setMode] = useState<'move' | 'delete'>(
    targets.length ? 'move' : 'delete',
  );
  const [targetID, setTargetID] = useState(targets[0]?.id ?? '');
  const [deleting, setDeleting] = useState(false);
  const [error, setError] = useState(false);
  useEffect(() => {
    if (category) {
      const nextTargets = categories.filter((item) => item.id !== category.id);
      setMode(nextTargets.length ? 'move' : 'delete');
      setTargetID(nextTargets[0]?.id ?? '');
      setError(false);
    }
  }, [categories, category]);
  if (!category) return null;
  const confirm = async () => {
    if (deleting || (mode === 'move' && !targetID)) return;
    setDeleting(true);
    setError(false);
    try {
      await deleteCategory(category.id, hasLinks ? mode : 'delete', targetID);
      onDeleted();
    } catch {
      setError(true);
    } finally {
      setDeleting(false);
    }
  };
  return (
    <Dialog
      open
      title={messages.categories.deleteTitle(category.name)}
      description={messages.categories.deleteDescription(category.links.length)}
      onClose={deleting ? () => undefined : onClose}
      footer={
        <>
          <Button disabled={deleting} onClick={onClose}>
            {messages.categories.cancel}
          </Button>
          <Button
            variant="danger"
            loading={deleting}
            disabled={mode === 'move' && !targetID}
            onClick={() => void confirm()}
          >
            {messages.categories.confirmDelete}
          </Button>
        </>
      }
    >
      <div class="category-delete">
        {hasLinks ? (
          <fieldset>
            <legend>{messages.categories.linkHandling}</legend>
            {targets.length > 0 ? (
              <label>
                <input
                  type="radio"
                  name="category-delete-mode"
                  checked={mode === 'move'}
                  onChange={() => setMode('move')}
                />
                <span>{messages.categories.moveLinks}</span>
              </label>
            ) : null}
            {mode === 'move' ? (
              <select
                aria-label={messages.categories.moveTarget}
                value={targetID}
                onChange={(event) => setTargetID(event.currentTarget.value)}
              >
                {targets.map((target) => (
                  <option key={target.id} value={target.id}>
                    {target.name}
                  </option>
                ))}
              </select>
            ) : null}
            <label>
              <input
                type="radio"
                name="category-delete-mode"
                checked={mode === 'delete'}
                onChange={() => setMode('delete')}
              />
              <span>
                {messages.categories.deleteLinks(category.links.length)}
              </span>
            </label>
          </fieldset>
        ) : null}
        {error ? <p role="alert">{messages.categories.deleteFailed}</p> : null}
      </div>
    </Dialog>
  );
}
