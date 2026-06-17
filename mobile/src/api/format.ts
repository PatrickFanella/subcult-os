import type { PricingMode } from '@/api/types';

export function formatDateTime(value: string) {
  const date = new Date(value);
  return Number.isNaN(date.getTime())
    ? value
    : new Intl.DateTimeFormat([], { dateStyle: 'medium', timeStyle: 'short' }).format(date);
}

export function formatDate(value: string) {
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? value : new Intl.DateTimeFormat([], { month: 'short', day: 'numeric' }).format(date);
}

export function formatTime(value: string) {
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? '' : new Intl.DateTimeFormat([], { timeStyle: 'short' }).format(date);
}

export function formatCurrency(cents: number, currency: string) {
  return new Intl.NumberFormat([], { style: 'currency', currency: currency.toUpperCase() }).format(cents / 100);
}

export function pricingLabel(pricingMode: PricingMode, cents: number, currency: string) {
  return pricingMode === 'free' ? 'Free' : formatCurrency(cents, currency);
}
