type BrandProps = { className?: string };

// The mark is the Studio pack lettermark; the wordmark stays live Space Mono text so it follows the theme.
export function Brand({ className = '' }: BrandProps) {
  return (
    <span className={`inline-flex shrink-0 items-center gap-3 whitespace-nowrap ${className}`}>
      <img src="/brand/mark.svg" alt="" width={36} height={36} className="size-9 shrink-0" />
      <span className="text-[1.375rem] font-bold normal-case leading-none tracking-normal text-fg-primary">Subcult OS</span>
    </span>
  );
}
