import { useEffect, useState } from 'preact/hooks';

import { messages } from '../i18n/messages';
import type { NavigationCategory } from '../navigation/types';
import { Button, Dialog } from '../ui/primitives';
import { deleteCategory } from './category-api';

type Props = {
  categories: readonly NavigationCategory[];
  category: NavigationCategory | null;
  onClose: () => void;
  onDeleted: () => void;
};

export function CategoryDeleteDialog({
  categories,
  category,
  onClose,
  onDeleted,
}: Props) {
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
