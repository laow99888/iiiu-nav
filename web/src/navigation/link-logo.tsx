import type { LogoTone } from './types';
import { useEffect, useState } from 'preact/hooks';

type LinkLogoProps = {
  label: string;
  text: string;
  tone: LogoTone;
  url?: string | null;
};

export function LinkLogo({ label, text, tone, url = null }: LinkLogoProps) {
  const [imageFailed, setImageFailed] = useState(false);
  useEffect(() => setImageFailed(false), [url]);
  return (
    <span
      class={`link-logo link-logo--${tone}`}
      aria-label={`${label} 标识`}
      role="img"
    >
      {url && !imageFailed ? (
        <img
          src={url}
          alt=""
          role="presentation"
          onError={() => setImageFailed(true)}
        />
      ) : (
        text
      )}
    </span>
  );
}
