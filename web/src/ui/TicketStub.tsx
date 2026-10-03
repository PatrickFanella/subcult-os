import type { ReactNode } from 'react';

type TicketStubProps = {
  kicker: string;
  children: ReactNode;
  // Tilt only display stubs. Anything a scanner or a reader must read stays square.
  tilt?: boolean;
  className?: string;
};

// The purple admit-one slab from the Subcult OS artwork. Pass the heading element as children so the
// page keeps its own heading level; the barcode is decoration.
export function TicketStub({ kicker, children, tilt = false, className = '' }: TicketStubProps) {
  return (
    <div className={`ticket-stub ${tilt ? 'ticket-stub-tilt' : ''} ${className}`}>
      <span className="ticket-stub-kicker">{kicker}</span>
      {children}
      <span className="ticket-bars" aria-hidden="true" />
    </div>
  );
}

type IndexListProps = { items: readonly string[]; className?: string };

// Numbered stages in the "01 / PLAN" form.
export function IndexList({ items, className = '' }: IndexListProps) {
  return (
    <ol className={`flex flex-col gap-4 ${className}`}>
      {items.map((item, index) => (
        <li key={item} className="index-label">{`${String(index + 1).padStart(2, '0')} / ${item}`}</li>
      ))}
    </ol>
  );
}

export function indexLabel(position: number, label: string) {
  return `${String(position).padStart(2, '0')} / ${label}`;
}
