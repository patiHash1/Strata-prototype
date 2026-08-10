# CSS

Strata uses **Templ Native CSS** (scoped inline styles compiled at build time).

All component styles are defined via `css` blocks in `.templ` files and compiled to Go code by `templ generate`. Styles are rendered inline within each HTML response — no external CSS stylesheet is required.

## Key benefits

- **Zero build dependencies** — no Node, npm, Tailwind CLI, or PostCSS
- **Type-safe** — CSS class names are Go functions that fail-fast at compile time
- **Scoped** — each `css` block generates unique class names, eliminating style collisions
- **All-in-Go** — templ compiles everything into the binary, no runtime CSS generation

## How to add styles

Define a `css` block in your `.templ` file:

```templ
css myComponentStyle() {
    background-color: #ffffff;
    border-radius: 0.75rem;
    padding: 1rem;
    &:hover {
        background-color: #f3f4f6;
    }
    @media (min-width: 1024px) {
        padding-left: 16rem;
    }
}

templ MyComponent() {
    <div class={ myComponentStyle() }>
        Hello, world!
    </div>
}
```

Run `templ generate` to regenerate Go code.
