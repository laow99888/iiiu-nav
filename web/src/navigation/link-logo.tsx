import type { LogoTone } from './types';
import { messages } from '../i18n/messages';
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
      aria-label={messages.navigation.linkLogoLabel(label)}
      role="img"
    >
      {url && !imageFailed ? (
        <img
          src={url}
          alt=""
          role="presentation"
          loading="lazy"
          decoding="async"
          onError={() => setImageFailed(true)}
        />
      ) : (
        text
      )}
    </span>
  );
}
