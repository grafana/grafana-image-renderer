# ADR-0004: Chrome

## Context and Problem Statement

We need a way to render Grafana dashboards as websites to PDFs and PNGs.

## Considered Options

- Draw the dashboards ourselves using a graphics library.
- Use Google Chrome (or Chromium, or equivalent Chrome DevTools Protocol browser) to render the dashboards.
- Use Mozilla Firefox (or equivalent WebDriver Bi-Di Protocol browser) to render the dashboards.
- Deprecate the feature.

## Decision Outcome

We're going for Chrome DevTools Protocol compliant browsers, primarily Chromium or Google Chrome:

- We do not think it is a wise investment to re-implement all panels and rendering logic.
- There are currently no good WebDriver Bi-Di clients around in Go.
- We cannot deprecate the feature, as it is widely used, and we have customers depending on it. This includes ourselves.
- Chromium is a stable browser with good headless support, and wide adoption.
- Chromium regularly patches security vulnerabilities.
- There are Chrome DevTools Protocol clients for Go that are mature.
