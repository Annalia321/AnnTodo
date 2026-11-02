import type { Card } from '@/api'
import { accentPresets } from '@/stores/settings'

export const cardColors = [...accentPresets, '#6D4C41', '#546E7A']

export const defaultCardColor = '#78909C'

export function cardColor(c: Pick<Card, 'color'> | null | undefined) {
  return c?.color && /^#[0-9a-f]{6}$/i.test(c.color) ? c.color : defaultCardColor
}

/** Picks a preset color for a new card, cycling through the palette. */
export function nextCardColor(count: number) {
  return cardColors[count % cardColors.length]
}
