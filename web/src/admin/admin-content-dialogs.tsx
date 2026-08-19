import {
  CategoryManager,
  type CategoryManagerAction,
} from '../categories/category-manager';
import { messages } from '../i18n/messages';
import { LinkDeleteDialog } from '../links/link-delete-dialog';
import { LinkEditor } from '../links/link-editor';
import { LinkOrderManager } from '../links/link-order-manager';
import type { NavigationCategory, NavigationLink } from '../navigation/types';
import { useToast } from '../ui/primitives';

export type CategoryManagerState =
  CategoryManagerAction | { type: 'list' } | null;

export type LinkEditorState = {
  categoryID: string;
  link: NavigationLink | null;
} | null;

type Props = {
  categories: readonly NavigationCategory[];
  categoryManager: CategoryManagerState;
  deleteLink: NavigationLink | null;
  linkEditor: LinkEditorState;
  linkOrderOpen: boolean;
  onCloseCategoryManager: () => void;
  onCloseDeleteLink: () => void;
  onCloseLinkEditor: () => void;
  onCloseLinkOrder: () => void;
  onRefresh: () => void;
};

export function AdminContentDialogs(props: Props) {
  const {
    categories,
    categoryManager,
    deleteLink,
    linkEditor,
    linkOrderOpen,
    onCloseCategoryManager,
    onCloseDeleteLink,
    onCloseLinkEditor,
    onCloseLinkOrder,
    onRefresh,
  } = props;
  const toast = useToast();

  const notifyLinkDeleted = () => {
    toast.notify({
      tone: 'success',
      title: messages.admin.linkDeleted,
      message: messages.admin.linkDeletedDescription,
    });
  };

  return (
    <>
      <CategoryManager
        key={categoryManagerKey(categoryManager)}
        open={categoryManager !== null}
        categories={categories}
        initialAction={
          categoryManager?.type === 'list' ? null : categoryManager
        }
        onClose={onCloseCategoryManager}
        onSaved={() => {
          onRefresh();
          if (categoryManager?.type === 'delete') {
            toast.notify({
              tone: 'success',
              title: messages.admin.categoryDeleted,
              message: messages.admin.categoryDeletedDescription,
            });
          } else if (categoryManager && categoryManager.type !== 'list') {
            toast.notify({
              tone: 'success',
              title: messages.admin.categorySaved,
              message: messages.admin.categorySavedDescription,
            });
          }
        }}
      />
      <LinkEditor
        key={`${linkEditor?.link?.id ?? 'new'}-${linkEditor?.categoryID ?? ''}`}
        open={linkEditor !== null}
        categories={categories}
        categoryID={linkEditor?.categoryID ?? ''}
        link={linkEditor?.link}
        onClose={onCloseLinkEditor}
        onSaved={() => {
          onCloseLinkEditor();
          toast.notify({
            tone: 'success',
            title: messages.admin.linkSaved,
            message: messages.admin.linkSavedDescription,
          });
          onRefresh();
        }}
        onDeleted={() => {
          onCloseLinkEditor();
          notifyLinkDeleted();
          onRefresh();
        }}
      />
      <LinkDeleteDialog
        link={deleteLink}
        onClose={onCloseDeleteLink}
        onDeleted={() => {
          onCloseDeleteLink();
          notifyLinkDeleted();
          onRefresh();
        }}
      />
      <LinkOrderManager
        open={linkOrderOpen}
        categories={categories}
        initialCategoryID={categories[0]?.id ?? ''}
        onClose={onCloseLinkOrder}
        onSaved={() => {
          onCloseLinkOrder();
          onRefresh();
        }}
      />
    </>
  );
}

function categoryManagerKey(state: CategoryManagerState) {
  if (state && 'categoryID' in state) {
    return `${state.type}-${state.categoryID}`;
  }
  return state?.type ?? 'closed';
}
