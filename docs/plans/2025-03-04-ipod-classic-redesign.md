# Podpal iPod Classic Era Redesign

**Date:** 2025-03-04  
**Theme:** iPod Classic Era (2001-2007)

## Overview

Redesign of the Podpal web interface to evoke the iPod Classic era aesthetic — the design language Apple used during the original iPod's reign. This creates a thematically appropriate experience for a tool that patches iPod firmware.

## Design Principles

- **Industrial elegance** — Precision engineering aesthetic
- **Generous whitespace** — Clean, breathing room between elements
- **Rounded corners** — Soft, approachable modernism
- **Monochromatic base** — Black, white, and grays with selective color
- **Subtle depth** — Shadows and layering without heaviness

## Color Palette

| Token | Value | Usage |
|-------|-------|-------|
| Background | `#f5f5f7` | Page background |
| Card | `#ffffff` | Content cards |
| Primary Text | `#1d1d1f` | Headlines, body text |
| Secondary Text | `#86868b` | Labels, hints, captions |
| Accent Blue | `#0066cc` | Links, focus states |
| Button | `#1d1d1f` | Primary actions |
| Button Hover | `#000000` | Button hover state |
| Success | `#34c759` | Success messages |
| Error | `#ff3b30` | Error messages |
| Warning BG | `#fff9e5` | Warning box background |
| Warning Border | `#ffd60a` | Warning accent |

## Typography

- **Font Stack:** `-apple-system, BlinkMacSystemFont, "SF Pro Text", "Segoe UI", sans-serif`
- **Title:** 48px, weight 600, letter-spacing -0.5px
- **Subtitle:** 21px, weight 400, color secondary
- **Body:** 17px, weight 400, line-height 1.47059
- **Labels:** 14px, weight 600, uppercase, letter-spacing 0.5px, secondary color
- **Small:** 12px, weight 400

## Components

### Main Card
- Background: white
- Border-radius: 18px
- Box-shadow: `0 4px 16px rgba(0,0,0,0.04)`
- Padding: 48px
- Max-width: 600px, centered

### Form Inputs
- Height: 48px
- Border-radius: 10px
- Border: 1px solid `#d2d2d7`
- Font-size: 17px
- Focus: border `#0066cc` with subtle glow

### Submit Button
- Background: `#1d1d1f`
- Color: white
- Full-width
- Height: 48px
- Border-radius: 10px
- Font: 17px, weight 600

### Warning Box
- Background: `#fff9e5`
- Border: 1px solid `#ffd60a`
- Left border: 4px solid `#ffd60a`
- Border-radius: 10px

### Explainer Section
- Generous spacing (32px between sections)
- Clean hierarchical typography
- Optional accordion for collapsibility

## Layout

Single-column centered layout:
1. Logo/Title at top
2. Warning box
3. Form with model selector and file upload
4. Preview area
5. Submit button
6. Divider
7. How it works section
8. Footer

## Files Modified

- `static/style.css` — Complete redesign
- `templates/index.html` — Minor structural adjustments

## Implementation Notes

- Preserve all existing HTMX functionality
- Maintain responsive behavior
- Keep all form field names for backend compatibility
- Add smooth transitions for interactive elements
