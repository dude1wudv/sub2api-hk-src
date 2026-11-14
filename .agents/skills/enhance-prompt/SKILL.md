---
name: enhance-prompt
description: >-
  Turn vague frontend requests into concrete UI requirements for new small apps,
  pages, layouts, and visual improvements. Use automatically for requests such
  as "做个小工具", "新增页面", or "优化界面", preserving the existing framework
  and design system. Continue implementation when the user asked to build or
  edit; return only a rewritten prompt when that is the request. Do not trigger
  for backend-only tasks or precise behavior-only bug fixes. Stitch is optional.
---

# Enhance UI Requirements and Prompts

Clarify frontend intent before implementing it or preparing a design prompt.
The local integration rules below take precedence over generic upstream examples.

## Local Integration Rules

- Detect the target frontend package and preserve its framework, router, API/state
  layer, i18n, existing components, package manager, and user edits. For Sub2API,
  default to `frontend/` and retain Vue/Pinia rather than introducing React.
- For a new app, use workspace conventions and the user's requested scope. Do not
  choose React, shadcn, Tailwind, or a new dependency solely because an example uses it.
- Use reasonable defaults for minor gaps. Ask only when an unanswered decision
  materially changes the deliverable and cannot be resolved from project evidence.
- For build/edit requests, keep the UI brief concise and continue implementation
  and the project's relevant validation. Do not stop at a copyable prompt.
- Do not generate documentation, placeholder metrics, or mock business claims,
  redesign unrelated screens, or add an autonomous page-generation loop.
- No Stitch account, MCP, or cloud upload is required. Consult the official
  prompting guide only for a task explicitly targeting Stitch:
  https://stitch.withgoogle.com/docs/learn/prompting/
- Use the host's available tools; resolve bundled resources from this skill's
  own directory. Match the user's language for output.

## When to Use This Skill

Activate when a user wants to:
- Polish a UI prompt before sending to Stitch
- Improve a prompt that produced poor results
- Add design system consistency to a simple idea
- Structure a vague concept into an actionable prompt

## Enhancement Pipeline

Follow these steps to enhance any prompt:

### Step 1: Assess the Input

Evaluate what's missing from the user's prompt:

| Element | Check for | If missing... |
|---------|-----------|---------------|
| **Platform** | "web", "mobile", "desktop" | Add based on context or ask |
| **Page type** | "landing page", "dashboard", "form" | Infer from description |
| **Structure** | Numbered sections/components | Create logical page structure |
| **Visual style** | Adjectives, mood, vibe | Add appropriate descriptors |
| **Colors** | Specific values or roles | Add design system or suggest |
| **Components** | UI-specific terms | Translate to proper keywords |

### Step 2: Check for DESIGN.md

Resolve the target frontend package first. Prefer its `.stitch/DESIGN.md`;
if an authoritative DESIGN.md already exists elsewhere in that package, reuse it.
Do not apply the umbrella workspace's or a sibling app's visual identity blindly.

**If a design document exists:**
1. Read it and check the relevant implementation for current tokens and components.
2. Include its palette, typography, and component conventions in the UI brief.
3. Note conflicts rather than silently overwriting the document or existing styles.

**If no design document exists:**
1. Inspect the package's relevant existing styles; use `stitch-extract-design-md`
   only when a meaningful style audit is needed, without requiring a new document.
2. For a greenfield app, choose a restrained design consistent with the user's
   requirements. Do not invent an existing brand or require unavailable skills.
3. Create `.stitch/DESIGN.md` only if the user asks to persist a design system.

### Step 3: Apply Enhancements

Transform the input using these techniques:

#### A. Add UI/UX Keywords

Replace vague terms with specific component names:

| Vague | Enhanced |
|-------|----------|
| "menu at the top" | "navigation bar with logo and menu items" |
| "button" | "primary call-to-action button" |
| "list of items" | "card grid layout" or "vertical list with thumbnails" |
| "form" | "form with labeled input fields and submit button" |
| "picture area" | "hero section with full-width image" |

#### B. Amplify the Vibe

Add descriptive adjectives to set the mood:

| Basic | Enhanced |
|-------|----------|
| "modern" | "clean, minimal, with generous whitespace" |
| "professional" | "sophisticated, trustworthy, with subtle shadows" |
| "fun" | "vibrant, playful, with rounded corners and bold colors" |
| "dark mode" | "dark theme with high-contrast accents on deep backgrounds" |

#### C. Structure the Page

Organize content into numbered sections:

```markdown
**Page Structure:**
1. **Header:** Navigation with logo and menu items
2. **Hero Section:** Headline, subtext, and primary CTA
3. **Content Area:** [Describe the main content]
4. **Footer:** Links, social icons, copyright
```

#### D. Format Colors Properly

When colors are mentioned, format them as:
```
Descriptive Name (#hexcode) for functional role
```

Examples:
- "Deep Ocean Blue (#1a365d) for primary buttons and links"
- "Warm Cream (#faf5f0) for page background"
- "Soft Gray (#6b7280) for secondary text"

### Step 4: Format the Output

Structure the enhanced prompt in this order:

```markdown
[One-line description of the page purpose and vibe]

**DESIGN SYSTEM (REQUIRED):**
- Platform: [Web/Mobile], [Desktop/Mobile]-first
- Theme: [Light/Dark], [style descriptors]
- Background: [Color description] (#hex)
- Primary Accent: [Color description] (#hex) for [role]
- Text Primary: [Color description] (#hex)
- [Additional design tokens...]

**Page Structure:**
1. **[Section]:** [Description]
2. **[Section]:** [Description]
...
```

## Output Options

**Implementation request:** Use the enhanced brief as an internal working aid,
then complete the requested UI change and its relevant checks.

**Prompt-only request:** Return the enhanced prompt as copyable text.

**Optional file output:** Write a prompt or design document only when explicitly
requested. Do not create a loop baton file or invoke `stitch-loop` automatically.

## Examples

### Example 1: Vague Input → Structured Prompt

**User input:**
> "make me a login page"

**Enhanced output:**
```markdown
A clean, trustworthy login page with a centered form and subtle branding.

**DESIGN SYSTEM (REQUIRED):**
- Platform: Web, Desktop-first
- Theme: Light, minimal, professional
- Background: Clean White (#ffffff)
- Surface: Soft Gray (#f9fafb) for form card
- Primary Accent: Deep Blue (#2563eb) for submit button and links
- Text Primary: Near Black (#111827) for headings
- Text Secondary: Medium Gray (#6b7280) for labels
- Buttons: Subtly rounded (8px), full-width on form
- Cards: Gently rounded (12px), soft shadow for elevation

**Page Structure:**
1. **Header:** Minimal logo, centered
2. **Login Card:** Centered form with email, password fields, "Forgot password?" link
3. **Submit Button:** Primary blue "Sign In" button
4. **Footer:** "Don't have an account? Sign up" link

---
For consistency across screens, reuse the current project's design tokens.
If a persisted design system is requested, use `stitch-extract-design-md`.
```

### Example 2: Feature Request → Targeted Edit

**User input:**
> "add a search bar"

**Enhanced output:**
```markdown
Add a search bar to the header navigation.

**Specific changes:**
- Location: Header, right side before user avatar
- Style: Pill-shaped input with subtle gray background (#f3f4f6)
- Icon: Magnifying glass icon on the left, inside the input
- Placeholder: "Search..." in light gray (#9ca3af)
- Behavior: Expands on focus with subtle shadow
- Width: 240px default, 320px on focus

**Context:** This is a targeted edit. Make only this change while preserving all existing elements.
```

## Tips for Best Results

1. **Be specific early** — Vague inputs need more enhancement
2. **Match the user's intent** — Don't over-design if they want simple
3. **Keep it structured** — Numbered sections help Stitch understand hierarchy
4. **Include the design system** — Consistency is key for multi-page projects
5. **One change at a time for edits** — Don't bundle unrelated changes
