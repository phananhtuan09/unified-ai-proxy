# Pre-defined Themes

This folder contains pre-defined themes for the `frontend-design-theme-factory` skill.

## Available Themes

| Theme | Personality | Best For |
|-------|-------------|----------|
| `professional-blue` | Corporate, trustworthy | Enterprise SaaS, fintech, B2B |
| `minimal-monochrome` | Sophisticated, editorial | Portfolios, luxury, content-first |
| `bold-vibrant` | Creative, energetic | Agencies, music, fashion |
| `bold-gradient` | Modern, tech-forward | SaaS, web3, developer tools |
| `warm-organic` | Natural, friendly | Food & beverage, artisan, blogs |
| `warm-earthy` | Grounded, comfortable | Health & wellness, sustainability |
| `creative-vibrant` | Artistic, expressive | Creative agencies, art platforms |
| `playful-colorful` | Fun, youthful | Kids apps, gaming, social |
| `digital-lavender` | Calming, AI-friendly | AI products, wellness, long-session SaaS |

## Theme Structure

Each theme JSON file contains:

- `name` - Display name
- `personality` - Keywords describing the aesthetic
- `use_cases` - Example product categories
- `colors`
  - `primary` - Main action color scale (50–950)
  - `secondary` - Supporting color scale (50–950)
  - `accent` - Highlight/contrast color scale (50–950) *(some themes)*
  - `neutral` - Text, backgrounds, borders (50–950)
  - `semantic` - success/warning/error/info with `_dark` variants
- `typography` - Font families (heading/body/mono), px scale, weight, lineHeight
- `spacing` - Named scale (xs → 4xl)
- `borderRadius` - none, sm → 2xl, full
- `shadows` - sm → 2xl (hue-tinted where appropriate)

## Usage

The theme-factory skill presents these themes when users ask for help choosing colors/fonts.

To add a custom theme:
1. Create a new `.theme.json` file in this folder
2. Follow the existing structure
3. Use descriptive personality keywords
4. Include `950` shade on all color scales (required for dark mode)
