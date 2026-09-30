# Subcult design system

Subcult uses the current mobile discovery and event screens as its visual
foundation: white surfaces, black actions, bold sans-serif headings, rounded
controls, and large event artwork. Event imagery supplies identity and color;
the interface keeps information and next steps readable. This direction was
selected on September 29, 2026.

## Foundations

`contracts/design/tokens.json` is the shared source. `color` defines the light
palette; `darkColor` defines its dark counterpart with the same semantic roles. Run
`node scripts/design-tokens.mjs` after editing it. The generator writes Tailwind
theme files for both clients and native TypeScript tokens. `make verify` checks
for drift and checks normal-text contrast for the admitted foreground/surface
pairs in both themes. Generated files stay committed so either app can build independently.

| Foundation | Rule |
| --- | --- |
| Surfaces | Light: soft neutral canvas and white panels. Dark: near-black canvas and charcoal panels. Immersive artwork and scanner surfaces stay black in both modes. |
| Text | Foreground roles adapt to the theme. Inverse text follows the primary action; on-immersive text stays white for artwork and scanner surfaces. |
| Actions | Black primary in light mode, white primary in dark mode, outlined secondary, quiet ghost. Use the outlined variant when a control needs a visible boundary. Every action needs a clear verb and a visible focus state. |
| Status | Green success, amber attention, red failure, blue information. Always include words; color alone cannot describe a state. |
| Typography | Platform sans-serif on native; Arial/Helvetica Neue on web. Heavy, tight headings; ordinary case and readable line height for instructions. No serif display theme. |
| Spacing | 4, 8, 12, 16, 24, 32, 48 px. Use 16 px page gutters on small screens and 24–32 px on wider screens. |
| Corners | 16 px controls, 24 px cards/insets, 28 px panels, 32 px artwork heroes. Pills are reserved for badges and compact navigation. |
| Targets | 48 px minimum for primary controls; 56 px fields and door controls. Keep scanner, navigation, and compact icon targets independently reviewable on devices. |
| Motion | Short state transitions; web respects reduced-motion preference. Do not animate information required to operate the door. |

Artwork must come from the event when available. Existing fallback imagery is
retained; the gallery uses one of the mobile app's existing fallback images.
Keep type on a solid area or a sufficiently dark scrim, and provide descriptive
alt text for meaningful web images. Do not use gradients or colored glass as a
replacement for event identity.

## Components and adoption

Web foundations live in `web/src/styles.css` and `web/src/ui/`. `Button` exposes
primary, secondary, and ghost variants, defaults to `type="button"`, and disables
busy actions. `Notice` announces errors as alerts and other feedback as status.
Use a native label with the shared `field` class for inputs. The existing
`publicUi` exports remain the common styling contract for public pages, tickets,
participant journeys, and the door.

The former dark operator screens now use semantic light-theme classes, including
workspace, event editing, identity, invitations, imports, lifecycle notices,
finance, and archive approval. Existing workflow handlers and confirmation steps
remain intact. Authentication uses the shared button and notice components.

Native foundations live in `mobile/src/theme/` and `mobile/src/global.css`.
Screen components read `useThemeTokens()` and `useThemedStyles()` so common
colors and corner sizes update when appearance changes; Uniwind
components use the generated theme. `PrimaryButton` exposes disabled and busy
states and button accessibility semantics. `Pill` supports all feedback tones.
The bottom navigation exposes its selected tab state and uses readable muted
labels. Immersive discovery and scanner screens keep their existing dark artwork
treatment. Screen-specific dimensions and image overlays remain local where
they serve a particular layout.

For new work, use semantic roles rather than adding another literal palette.
Extract a component when interaction or structure is repeated; do not build a
second router, form framework, or token package around these foundations.

## Appearance

Both clients default to the system setting and offer Light, Dark, and System.
The web Appearance selector is at the top of the page. Native preferences are
in Settings → Appearance. Each client saves its local preference; it is not an
account setting and does not sync across devices. Storage failures leave the
current session's controls usable.

Web reads its saved setting before rendering, responds to system changes in
System mode, and syncs saved preference changes between tabs. Native uses the
installed Uniwind theme API for both utility classes and reactive StyleSheet
factories. Fixed white scrims and inverse text on artwork stay independent of
the working-surface palette.

Base link and font resets belong in Tailwind's base layer. An unlayered
`a { color: inherit }` overrides text-color utilities and can make a light button
inside a dark artwork panel unreadable. Verify actual computed colors as well
as token contrast. Mode changes switch text and backgrounds together; controls
only transition opacity, transforms, and shadows.

## Review and verification

Start the isolated preview with `bash scripts/dev-env.sh start`. Open
`/design-system` on its printed URL to review event treatment, type, controls,
disabled/busy states, fields, badges, and feedback. The gallery is a development
route and does not appear in product navigation. Its Save/Reset controls affect
only local gallery state.

Run `bash scripts/dev-env.sh verify` for the repository checks and disposable DB
gate. `make check-contracts` includes token drift and contrast checks. An Expo
export checks native bundling; it does not establish visual quality or touch
behavior on a device.

Before release, review the gallery and real discovery, event, authentication,
workspace, editor, ticket, and door journeys at narrow and wide web widths.
Review discovery, event details, forms, navigation, and scanner on iOS and
Android, including keyboard focus where supported and increased text size.
Build and unit-test results are separate from browser and device evidence.
