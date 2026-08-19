import {
  draggable,
  dropTargetForElements,
  monitorForElements,
} from '@atlaskit/pragmatic-drag-and-drop/adapter/element-adapter';
import type { ComponentChildren } from 'preact';
import { useEffect, useRef, useState } from 'preact/hooks';

import { messages } from '../../i18n/messages';
import { ArrowDown, ArrowUp, GripVertical } from '../icons/interface-icons';
import { Button } from '../primitives/button';
import { Tooltip } from '../primitives/tooltip';
import { reorderItems } from './reorder-items';

const sortableType = 'iiiu-nav-sortable';

type SortableItemProps = {
  children: ComponentChildren;
  disabled?: boolean;
  id: string;
  index: number;
  label: string;
  onMove: (fromIndex: number, toIndex: number) => void;
  total: number;
};

function SortableItem({
  children,
  disabled,
  id,
  index,
  label,
  onMove,
  total,
}: SortableItemProps) {
  const itemRef = useRef<HTMLLIElement>(null);
  const handleRef = useRef<HTMLButtonElement>(null);
  const [dragging, setDragging] = useState(false);
  const [over, setOver] = useState(false);

  useEffect(() => {
    const element = itemRef.current;
    const dragHandle = handleRef.current;
    if (!element || !dragHandle || disabled) {
      return;
    }

    const cleanupDraggable = draggable({
      element,
      dragHandle,
      getInitialData: () => ({ id, index, type: sortableType }),
      onDragStart: () => setDragging(true),
      onDrop: () => setDragging(false),
    });
    const cleanupDropTarget = dropTargetForElements({
      element,
      getData: () => ({ id, index, type: sortableType }),
      canDrop: ({ source }) => source.data.type === sortableType,
      onDragEnter: () => setOver(true),
      onDragLeave: () => setOver(false),
      onDrop: () => setOver(false),
    });

    return () => {
      cleanupDraggable();
      cleanupDropTarget();
    };
  }, [disabled, id, index]);

  return (
    <li
      ref={itemRef}
      class={`ui-sortable__item${dragging ? ' is-dragging' : ''}${over ? ' is-over' : ''}`}
      data-sortable-id={id}
    >
      <Tooltip content={messages.ui.dragItem(label)}>
        <button
          ref={handleRef}
          class="ui-sortable__handle"
          type="button"
          disabled={disabled}
          aria-label={messages.ui.dragItem(label)}
        >
          <GripVertical aria-hidden="true" />
        </button>
      </Tooltip>
      <div class="ui-sortable__content">{children}</div>
      <div class="ui-sortable__actions">
        <Button
          variant="ghost"
          size="small"
          icon={ArrowUp}
          disabled={disabled || index === 0}
          aria-label={messages.ui.moveItemUp(label)}
          onClick={() => onMove(index, index - 1)}
        />
        <Button
          variant="ghost"
          size="small"
          icon={ArrowDown}
          disabled={disabled || index === total - 1}
          aria-label={messages.ui.moveItemDown(label)}
          onClick={() => onMove(index, index + 1)}
        />
      </div>
    </li>
  );
}

type SortableListProps<T> = {
  disabled?: boolean;
  getId: (item: T) => string;
  getLabel: (item: T) => string;
  items: readonly T[];
  onReorder: (items: T[]) => void;
  renderItem: (item: T) => ComponentChildren;
};

export function SortableList<T>({
  disabled,
  getId,
  getLabel,
  items,
  onReorder,
  renderItem,
}: SortableListProps<T>) {
  const itemsRef = useRef(items);
  const onReorderRef = useRef(onReorder);

  useEffect(() => {
    itemsRef.current = items;
    onReorderRef.current = onReorder;
  }, [items, onReorder]);

  const move = (fromIndex: number, toIndex: number) => {
    if (toIndex < 0 || toIndex >= items.length || fromIndex === toIndex) {
      return;
    }
    onReorder(reorderItems(items, fromIndex, toIndex));
  };

  useEffect(() => {
    if (disabled) {
      return;
    }

    return monitorForElements({
      canMonitor: ({ source }) => source.data.type === sortableType,
      onDrop: ({ location, source }) => {
        const target = location.current.dropTargets[0];
        const fromIndex = source.data.index;
        const toIndex = target?.data.index;
        if (typeof fromIndex !== 'number' || typeof toIndex !== 'number') {
          return;
        }
        onReorderRef.current(
          reorderItems(itemsRef.current, fromIndex, toIndex),
        );
      },
    });
  }, [disabled]);

  return (
    <ul class="ui-sortable">
      {items.map((item, index) => (
        <SortableItem
          key={getId(item)}
          id={getId(item)}
          index={index}
          label={getLabel(item)}
          total={items.length}
          disabled={disabled}
          onMove={move}
        >
          {renderItem(item)}
        </SortableItem>
      ))}
    </ul>
  );
}
