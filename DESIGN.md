# Design Contract

Harbor should feel like a calm personal start page. These rules describe the
current direction and constrain future UI changes.

## Home

- Use a locally packaged full-screen background with enough contrast for text.
- Place a large clock and a quieter English date above a grid of service cards.
- Cards show an icon and name, with a short description or hostname as secondary
  text. Keep generous spacing and a consistent rounded shape.
- Use restrained iOS-inspired glass surfaces, soft borders, and minimal shadows.
- Put Settings in the top-right corner and a quiet All apps launcher at the bottom.
- Keep search, add/edit/delete controls, statistics, and management toolbars out
  of the default home view.
- An empty collection should explain how to add or choose apps without filling
  the home screen with onboarding panels.

## Panels and Actions

- Right-click opens All apps. The bottom launcher, Settings, and `/` provide
  alternate entry points; editable text retains its normal context menu.
- All apps contains the complete collection and a final add tile. Search narrows
  the collection. Editing controls and home selection live here.
- Settings contains appearance, clock, background, management entry points,
  and editing access. Keep it near its top-right trigger on larger screens.
- Add/edit forms use labeled fields, explicit save/cancel actions, an icon
  preview, and Show on home. Return users to the panel that opened the editor.
- Explain failures near the action and preserve the entered form values.
- Confirm deletion; hiding a card must never delete its service.

## Accessibility and Responsiveness

- Use semantic links for destinations and buttons for actions. Give icon-only
  buttons an accessible name and maintain visible keyboard focus.
- Keep dialogs bounded to the viewport, with scrollable content. Support Escape,
  close controls, and outside-click dismissal where appropriate.
- Keep essential actions available without hover or right-click. Design primary
  touch actions for a 44px target, even when the visible icon is smaller.
- Reflow cards on small screens without horizontal page overflow. Keep a useful
  two-column home grid on ordinary phones.
- Respect reduced-motion preferences and avoid continuous decorative animation.
- Use short, interruptible transform/opacity transitions for glass surfaces,
  with a softer entrance and a quicker exit. Keep blur radii and shadows static
  during motion; frost individual surfaces instead of the full-screen backdrop.
- Reuse unchanged service cards during filtering and updates so icons and focus
  remain stable. Searching All apps must not rebuild the home grid.
- Provide opaque glass fallbacks when backdrop filtering is unavailable or the
  browser requests reduced transparency.
- Use system fonts, local assets, and CSS variables for shared appearance roles.
  Do not introduce external font or icon requests for decoration.

## Reference and Ownership

The clock-first composition is inspired by Lime Start Page; Harbor uses cards
on home and a separate All apps panel. Asset provenance and the reference link
are recorded in [ASSETS.md](ASSETS.md).
