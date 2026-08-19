import {
  autoUpdate,
  computePosition,
  flip,
  offset,
  shift,
  type Placement,
} from '@floating-ui/dom';
import { useCallback, useEffect, useState } from 'preact/hooks';

type FloatingElements = {
  reference: HTMLElement | null;
  floating: HTMLElement | null;
};

export function useFloatingPosition(
  open: boolean,
  placement: Placement = 'bottom-start',
) {
  const [elements, setElements] = useState<FloatingElements>({
    reference: null,
    floating: null,
  });

  useEffect(() => {
    if (!open || !elements.reference || !elements.floating) {
      return;
    }

    const update = () => {
      void computePosition(elements.reference!, elements.floating!, {
        placement,
        strategy: 'fixed',
        middleware: [offset(6), flip({ padding: 8 }), shift({ padding: 8 })],
      }).then(({ x, y }) => {
        Object.assign(elements.floating!.style, {
          left: `${x}px`,
          top: `${y}px`,
        });
      });
    };

    return autoUpdate(elements.reference, elements.floating, update);
  }, [elements, open, placement]);

  const referenceRef = useCallback((element: HTMLElement | null) => {
    setElements((current) =>
      current.reference === element
        ? current
        : { ...current, reference: element },
    );
  }, []);

  const floatingRef = useCallback((element: HTMLElement | null) => {
    setElements((current) =>
      current.floating === element
        ? current
        : { ...current, floating: element },
    );
  }, []);

  return { referenceRef, floatingRef };
}
