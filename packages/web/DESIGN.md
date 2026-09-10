# Auth Gate interface context

Auth Gate is an infrastructure administration tool. Operational clarity and consistency with the existing control plane take priority over a visual redesign.

The maintained visual specification is [DESIGN_SYSTEM.md](DESIGN_SYSTEM.md). Runtime color, typography, spacing, themes and motion remain owned by `src/styles/tokens.css`, `src/styles/base.css`, `src/styles/components.css` and the existing Tailwind configuration. This file does not generate or duplicate those tokens.

## Localization

- Supported product locales: `en`, `zh-CN`, provided by `src/lib/i18n`.
- Application labels, validation, errors, accessible names, dates and numbers follow the selected language. Technical hostnames, paths, protocol names and HTTP methods preserve their values.
- Validate referenced translation keys as well as equality between locale resources: a key can be absent from both languages.
- Standalone gateway pages use their own origin's saved language, then browser preference; their controls and errors remain bilingual without a frontend bundle.

## Log workflow

Place filters before statistics because they define the dataset for the entire page. Keep the existing cards, typography, tokens and light/dark themes. Full dates and request hostnames distinguish similar paths across domains and days.

Shared controls and interaction ownership are recorded in [UX-CONTRACT.md](UX-CONTRACT.md). The desktop log table retains horizontal scrolling on narrow screens so every original diagnostic field stays accessible, without imposing a fixed-height ancestor on forms or dialogs.
