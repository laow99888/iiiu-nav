import { Compass } from '../ui/icons/interface-icons';

type SiteIdentityProps = {
  compact?: boolean;
  name: string;
  logoUrl?: string;
};

export function SiteIdentity({
  compact = false,
  logoUrl = '',
  name,
}: SiteIdentityProps) {
  return (
    <div class={`site-identity${compact ? ' site-identity--compact' : ''}`}>
      <span class="site-identity__mark" aria-hidden="true">
        {logoUrl ? <img src={logoUrl} alt="" /> : <Compass />}
      </span>
      <strong>{name}</strong>
    </div>
  );
}
