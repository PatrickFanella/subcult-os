import type { ButtonHTMLAttributes } from 'react';

type ButtonProps = ButtonHTMLAttributes<HTMLButtonElement> & {
  variant?: 'primary' | 'secondary' | 'ghost' | 'danger';
  busy?: boolean;
};
const variants = { primary: 'btn-primary', secondary: 'btn-secondary', ghost: 'btn-ghost', danger: 'btn-danger' };

export function Button({ variant = 'primary', busy = false, disabled, className = '', children, type = 'button', ...props }: ButtonProps) {
  return (
    <button {...props} type={type} disabled={disabled || busy} aria-busy={busy || undefined}
      className={`${variants[variant]} inline-flex items-center justify-center gap-2 px-5 py-3 text-sm transition ${className}`}>
      {children}
    </button>
  );
}
