import { Folder, type LucideProps } from 'lucide-preact';

import {
  categoryIconRegistry,
  type CategoryIconName,
} from './category-icon-registry';

type CategoryIconProps = LucideProps & {
  name?: CategoryIconName | null;
};

export function CategoryIcon({ name = 'folder', ...props }: CategoryIconProps) {
  const Icon = categoryIconRegistry[name ?? 'folder']?.component ?? Folder;
  return <Icon aria-hidden="true" focusable="false" {...props} />;
}
